package types

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	database "github.com/b-harvest/Harvestmon/database"
	"github.com/b-harvest/Harvestmon/log"
	gorm_mysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"os"
	"time"
)

func NewCheckerClient(cfg *CheckerConfig, alertDefinition *AlertDefinition, customAgentConfigs []CustomAgentConfig) (*CheckerClient, error) {
	err := os.Setenv("AWS_DEFAULT_REGION", "ap-northeast-2")
	if err != nil {
		return nil, err
	}

	// Fetch defaultConfig using env AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY
	awsConfig, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, err
	}

	var agentLevelList = make(map[AgentName]map[AlertName]AlertLevel)

	for _, alertLevel := range alertDefinition.AlertLevel {
		if agentLevelList[DEFAULT_AGENT_NAME] == nil {
			agentLevelList[DEFAULT_AGENT_NAME] = make(map[AlertName]AlertLevel)
		}
		agentLevelList[DEFAULT_AGENT_NAME][alertLevel.AlertName] = AlertLevel{
			AlertName:  alertLevel.AlertName,
			AlertLevel: alertLevel.AlertLevel,
		}
	}

	var alarmerList = make(map[AgentName]map[string][]Alarmer)
	for _, a := range alertDefinition.Alarmer {
		if a.AlarmResendDuration == nil {
			defaultResend := 5 * time.Minute
			a.AlarmResendDuration = &defaultResend
		}

		if alarmerList[DEFAULT_AGENT_NAME] == nil {
			alarmerList[DEFAULT_AGENT_NAME] = make(map[string][]Alarmer)
		}

		for _, targetLevel := range a.TargetLevels {
			alarmerList[DEFAULT_AGENT_NAME][targetLevel] = append(alarmerList[DEFAULT_AGENT_NAME][targetLevel], Alarmer{
				TargetLevels:        a.TargetLevels,
				AlarmerName:         a.AlarmerName,
				Format:              a.Format,
				AlarmParamList:      a.AlarmParamList,
				AlarmResendDuration: a.AlarmResendDuration,
			})
		}
	}

	for _, cac := range customAgentConfigs {
		for _, alertLevel := range cac.AlertLevel {
			agentLevelList[cac.AgentName][alertLevel.AlertName] = AlertLevel{
				AlertName:  alertLevel.AlertName,
				AlertLevel: alertLevel.AlertLevel,
			}
		}
		for _, a := range cac.Alarmer {
			if a.AlarmResendDuration == nil {
				defaultResend := 5 * time.Minute
				a.AlarmResendDuration = &defaultResend
			}
			for _, targetLevel := range a.TargetLevels {
				if alarmerList[cac.AgentName] == nil {
					alarmerList[cac.AgentName] = map[string][]Alarmer{}
				}
				alarmerList[cac.AgentName][targetLevel] = append(alarmerList[cac.AgentName][targetLevel], Alarmer{
					TargetLevels:        a.TargetLevels,
					AlarmerName:         a.AlarmerName,
					Format:              a.Format,
					AlarmParamList:      a.AlarmParamList,
					AlarmResendDuration: a.AlarmResendDuration,
				})
			}
		}
	}

	rpcClient := CheckerClient{
		DB:                  database.GetDatabase(&cfg.Database),
		LambdaClient:        lambda.NewFromConfig(awsConfig),
		AgentAlertLevelList: agentLevelList,
		AlarmerList:         alarmerList,
	}
	return &rpcClient, nil
}

// CheckerClient determines what alarmer should be used to send alarm associated with AlertLevelList.
type CheckerClient struct {
	DB *sql.DB
	// Key of AlertLevelList is same with AlertLevel.AlertName
	AgentAlertLevelList map[AgentName]map[AlertName]AlertLevel
	AlarmerList         map[AgentName]map[string][]Alarmer

	LambdaClient *lambda.Client
}

type AgentName string

func (c *CheckerClient) GetAlertLevelList(agentName AgentName, alertLevelName string) AlertLevel {
	var (
		alertLevel AlertLevel
		exists     bool
	)
	if alertLevel, exists = c.AgentAlertLevelList[agentName][AlertName(alertLevelName)]; exists {
	} else {
		alertLevel = c.AgentAlertLevelList[DEFAULT_AGENT_NAME][AlertName(alertLevelName)]
	}
	return alertLevel
}

func (c *CheckerClient) GetAlarmerList(agentName AgentName, alertLevel string) []Alarmer {
	if len(c.AlarmerList[agentName][alertLevel]) == 0 {
		return c.AlarmerList[DEFAULT_AGENT_NAME][alertLevel]
	} else {
		return c.AlarmerList[agentName][alertLevel]
	}
}

// InvokeLambda invokes the Lambda function specified by functionName, passing the parameters
// as a JSON payload. When getLog is true, types.LogTypeTail is specified, which tells
// Lambda to include the last few log lines in the returned result.
func (c *CheckerClient) InvokeLambda(functionName string, parameters any, getLog bool) *lambda.InvokeOutput {
	logType := types.LogTypeNone
	if getLog {
		logType = types.LogTypeTail
	}
	payload, err := json.Marshal(parameters)
	if err != nil {
		log.Error(errors.New(fmt.Sprintf("Couldn't marshal parameters to JSON. Here's why %v\n", err)))
	}
	invokeOutput, err := c.LambdaClient.Invoke(context.Background(), &lambda.InvokeInput{
		FunctionName: aws.String(functionName),
		LogType:      logType,
		Payload:      payload,
	})
	if err != nil {
		log.Error(errors.New(fmt.Sprintf("Couldn't invoke function %v. Here's why: %v\n", functionName, err)))
	}
	return invokeOutput
}

func (r *CheckerClient) GetDatabase() *gorm.DB {
	gormDB, err := gorm.Open(gorm_mysql.New(gorm_mysql.Config{Conn: r.DB}))
	if err != nil {
		panic(err)
	}
	return gormDB
}
