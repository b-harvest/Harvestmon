//go:build !local

package main

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func start() {
	lambda.Start(router)
}

func router(ctx context.Context, event json.RawMessage) (interface{}, error) {
	// Attempt to decode as an API Gateway event
	var apiEvent events.APIGatewayProxyRequest
	if err := json.Unmarshal(event, &apiEvent); err == nil && apiEvent.HTTPMethod != "" {
		// This is an API Gateway event
		return restHandler(ctx, apiEvent)
	}

	return nil, handleAction()
}
