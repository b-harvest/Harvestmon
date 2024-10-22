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
	"github.com/gorhill/cronexpr"
	gorm_mysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"os"
	"strings"
	"time"
)

func NewCheckerClient(cfg *CheckerConfig, alertDefinition *AlertDefinition, customAgentConfigs []CustomAgentConfig, wdb, rdb *sql.DB) (*CheckerClient, error) {
	client := &CheckerClient{}
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

	var agentLevels = make(map[AgentName]map[AlertName]AlertLevel)

	for _, alertLevel := range alertDefinition.AlertLevel {
		if agentLevels[DEFAULT_AGENT_NAME] == nil {
			agentLevels[DEFAULT_AGENT_NAME] = make(map[AlertName]AlertLevel)
		}
		agentLevels[DEFAULT_AGENT_NAME][alertLevel.AlertName] = AlertLevel{
			AlertName:  alertLevel.AlertName,
			AlertLevel: alertLevel.AlertLevel,
		}
	}

	var (
		alarmers    = make(map[AgentName]map[string][]Alarmer)
		snoozeCrons = make(map[AgentName][]SnoozeCron)
	)
	for _, a := range alertDefinition.Alarmer {
		if a.AlarmResendDuration == nil {
			defaultResend := 5 * time.Minute
			a.AlarmResendDuration = &defaultResend
		}

		if alarmers[DEFAULT_AGENT_NAME] == nil {
			alarmers[DEFAULT_AGENT_NAME] = make(map[string][]Alarmer)
		}

		for _, targetLevel := range a.TargetLevels {

			functionName := a.FunctionName
			if functionName == "" {
				functionName = a.AlarmerName
			}
			alarmers[DEFAULT_AGENT_NAME][targetLevel] = append(alarmers[DEFAULT_AGENT_NAME][targetLevel], Alarmer{
				TargetLevels:        a.TargetLevels,
				AlarmerName:         a.AlarmerName,
				FunctionName:        functionName,
				Format:              a.Format,
				AlarmParamList:      a.AlarmParamList,
				AlarmResendDuration: a.AlarmResendDuration,
			})
		}
	}

	for _, cac := range customAgentConfigs {
		// Prevent when there are no alert definition for custom Agent.
		agentLevels[cac.AgentName] = make(map[AlertName]AlertLevel)

		for _, alertLevel := range cac.AlertLevel {
			agentLevels[cac.AgentName][alertLevel.AlertName] = AlertLevel{
				AlertName:  alertLevel.AlertName,
				AlertLevel: alertLevel.AlertLevel,
			}
		}
		for _, level := range agentLevels[DEFAULT_AGENT_NAME] {
			if _, exists := agentLevels[cac.AgentName][level.AlertName]; !exists {
				agentLevels[cac.AgentName][level.AlertName] = agentLevels[DEFAULT_AGENT_NAME][level.AlertName]
			}
		}

		if alarmers[cac.AgentName] == nil {
			alarmers[cac.AgentName] = make(map[string][]Alarmer)
		}

		for _, a := range cac.Alarmer {
			if a.AlarmResendDuration == nil {
				defaultResend := 5 * time.Minute
				a.AlarmResendDuration = &defaultResend
			}
			for _, targetLevel := range a.TargetLevels {

				functionName := a.FunctionName
				if functionName == "" {
					functionName = a.AlarmerName
				}
				alarmers[cac.AgentName][targetLevel] = append(alarmers[cac.AgentName][targetLevel], Alarmer{
					TargetLevels:        a.TargetLevels,
					AlarmerName:         a.AlarmerName,
					FunctionName:        functionName,
					Format:              a.Format,
					AlarmParamList:      a.AlarmParamList,
					AlarmResendDuration: a.AlarmResendDuration,
				})
			}
		}

		for targetLevel, alarmerDefinitions := range alarmers[DEFAULT_AGENT_NAME] {
			if _, exists := alarmers[cac.AgentName][targetLevel]; !exists {
				alarmers[cac.AgentName][targetLevel] = alarmerDefinitions
			}
		}

		snoozeCrons[cac.AgentName] = cac.SnoozeCrons
	}

	if rdb == nil {
		rdb = wdb
	}

	client.RDB = rdb
	client.WDB = wdb
	client.LambdaClient = lambda.NewFromConfig(awsConfig)
	client.AgentAlertLevels = agentLevels
	client.Alarmers = alarmers
	client.AgentSnoozeCrons = snoozeCrons

	return client, nil
}

// CheckerClient determines what alarmer should be used to send alarm associated with AlertLevelList.
type CheckerClient struct {
	WDB *sql.DB
	RDB *sql.DB
	// Key of AlertLevelList is same with AlertLevel.AlertName
	AgentAlertLevels map[AgentName]map[AlertName]AlertLevel
	Alarmers         map[AgentName]map[string][]Alarmer
	AgentSnoozeCrons map[AgentName][]SnoozeCron

	LambdaClient *lambda.Client
}

type AgentName string

func (c *CheckerClient) GetAlertLevel(agentName AgentName, alertLevelKeyword ...string) *AlertLevel {

	var (
		containingStoredAlertLevels []AlertLevel
		resultAlertLevel            = new(AlertLevel)
	)

	for _, storedAlertLevel := range c.AgentAlertLevels[agentName] {
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
	if len(c.Alarmers[agentName][alertLevel]) == 0 {
		return c.Alarmers[DEFAULT_AGENT_NAME][alertLevel]
	} else {
		return c.Alarmers[agentName][alertLevel]
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
		FunctionName:   aws.String(functionName),
		LogType:        logType,
		Payload:        payload,
		InvocationType: types.InvocationTypeRequestResponse,
	})
	if err != nil {
		log.Error(errors.New(fmt.Sprintf("Couldn't invoke function %v. Here's why: %v\n", functionName, err)))
	}
	return invokeOutput
}

func (r *CheckerClient) GetWDatabase() *gorm.DB {
	gormDB, err := gorm.Open(gorm_mysql.New(gorm_mysql.Config{Conn: r.WDB}), &gorm.Config{Logger: nil})
	if err != nil {
		panic(err)
	}
	return gormDB
}

func (r *CheckerClient) GetRDatabase() *gorm.DB {
	gormDB, err := gorm.Open(gorm_mysql.New(gorm_mysql.Config{Conn: r.RDB}), &gorm.Config{Logger: nil})
	if err != nil {
		panic(err)
	}
	return gormDB
}

func (r *CheckerClient) IsSnooze(agentName AgentName) bool {
	if r.AgentSnoozeCrons[agentName] != nil {
		now := time.Now()
		for _, cron := range r.AgentSnoozeCrons[agentName] {
			beforeTime := time.Now().Add(-cron.Duration)
			var nextTime time.Time
			for nextTime.Before(now) {
				beforeTime = beforeTime.Add(cron.Duration)
				nextTime = cronexpr.MustParse(cron.StartCron).Next(beforeTime)
			}

			cronStart := cronexpr.MustParse(cron.StartCron).Next(beforeTime.Add(-cron.Duration))

			if now.After(cronStart) && now.Before(cronStart.Add(cron.Duration)) {
				return true
			}
		}
	}
	return false
}
