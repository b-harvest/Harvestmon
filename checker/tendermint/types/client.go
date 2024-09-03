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
	"strings"
	"time"
)

func NewCheckerClient(cfg *CheckerConfig, alertDefinition *AlertDefinition, customAgentConfigs []CustomAgentConfig) (*CheckerClient, error) {
	if os.Getenv(database.EnvDBAwsRegion) == "" {
		err := os.Setenv(database.EnvDBAwsRegion, "ap-northeast-2")
		if err != nil {
			return nil, err
		}
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
		// Prevent when there are no alert definition for custom Agent.
		agentLevelList[cac.AgentName] = make(map[AlertName]AlertLevel)

		for _, alertLevel := range cac.AlertLevel {
			agentLevelList[cac.AgentName][alertLevel.AlertName] = AlertLevel{
				AlertName:  alertLevel.AlertName,
				AlertLevel: alertLevel.AlertLevel,
			}
		}
		for _, level := range agentLevelList[DEFAULT_AGENT_NAME] {
			if _, exists := agentLevelList[cac.AgentName][level.AlertName]; !exists {
				agentLevelList[cac.AgentName][level.AlertName] = agentLevelList[DEFAULT_AGENT_NAME][level.AlertName]
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

	db, err := database.GetDatabase("resources/default_checker_rules.yaml")

	rpcClient := CheckerClient{
		DB:                  db,
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

func (c *CheckerClient) GetAlertLevel(agentName AgentName, alertLevelKeyword ...string) *AlertLevel {

	var (
		containingStoredAlertLevels []AlertLevel
		resultAlertLevel            = new(AlertLevel)
	)

	for _, storedAlertLevel := range c.AgentAlertLevelList[agentName] {
		var (
			contains = true
		)

		for _, singleAlertLevelFactor := range alertLevelKeyword {
			var found bool
			for _, sepStoredAlertLevelFactor := range strings.Split(string(storedAlertLevel.AlertName), ",") {
				if sepStoredAlertLevelFactor == singleAlertLevelFactor {
					found = true
					break
				}
			}
			if !found {
				contains = false
				break
			}
		}

		if contains {
			containingStoredAlertLevels = append(containingStoredAlertLevels, storedAlertLevel)
			resultAlertLevel = &storedAlertLevel
		}
	}
	if len(containingStoredAlertLevels) == 0 {
		return nil
	}

	for _, containingStoredAlertLevel := range containingStoredAlertLevels {
		if len(strings.Split(resultAlertLevel.AlertName.String(), ",")) > len(strings.Split(containingStoredAlertLevel.AlertName.String(), ",")) {
			resultAlertLevel = &containingStoredAlertLevel
		}
	}
	return resultAlertLevel
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
	gormDB, err := gorm.Open(gorm_mysql.New(gorm_mysql.Config{Conn: r.DB}), &gorm.Config{Logger: nil})
	if err != nil {
		panic(err)
	}
	return gormDB
}
