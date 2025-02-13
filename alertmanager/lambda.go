//go:build lambda

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func start() {
	lambda.Start(router)
}

func router(ctx context.Context, event json.RawMessage) (interface{}, error) {
	var apiEvent events.APIGatewayProxyRequest
	if err := json.Unmarshal(event, &apiEvent); err == nil && apiEvent.HTTPMethod != "" {
		return restHandler(ctx, apiEvent)
	}

	var snsEvent events.SNSEvent
	err := json.Unmarshal(event, &snsEvent)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal SNS event: %w", err)
	}

	err = handleAction(&snsEvent)
	if err != nil {
		return nil, err
	}
	return map[string]string{"status": "processed"}, nil
}
