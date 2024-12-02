package alarmer

import (
	"fmt"
	"github.com/b-harvest/Harvestmon/checker/ethereum/types"
	_const "github.com/b-harvest/Harvestmon/const"
	"github.com/b-harvest/Harvestmon/log"
	"github.com/b-harvest/Harvestmon/repository"
	"github.com/google/uuid"
	"reflect"
	"regexp"
	"strings"
	"time"
)

func RunAlarm(cfg *types.CheckerConfig, client types.CheckerClient, alert types.Alert) error {
	alertRecordRepository := repository.AlertRecordRepository{
		BaseRepository: repository.BaseRepository{
			DB:       *client.GetWDatabase(),
			CommitId: cfg.CommitId,
		},
	}

	now := time.Now().UTC()
	startTime := now.Add(-(*alert.Alarmer.AlarmResendDuration))

	// Check if an alert has already been sent
	result, err := alertRecordRepository.ExistsIfAlertRecordIsMarkedOrAlreadySent(
		alert.AlertLevel.AlertName.String(),
		alert.Alarmer.AlarmerName,
		string(alert.Agent),
		startTime, now, 30*time.Minute)
	if err != nil {
		return fmt.Errorf("error checking alert record: %w", err)
	}

	if client.IsSnooze(alert.Agent) || result {
		log.Info(aprintf("Alert has already sent to target within %v or Marked by operator. agent: %s, alert: %s", alert.Alarmer.AlarmResendDuration, alert.Agent, alert.AlertLevel.AlertName))
		return nil
	}

	// Prepare the payload for the alarm
	payload := make(map[string]any)
	alarmMap := map[string]string{
		"AGENT":         string(alert.Agent),
		"ALERT_NAME":    string(alert.AlertLevel.AlertName),
		"ALERT_LEVEL":   alert.AlertLevel.AlertLevel,
		"ALERT_SERVICE": _const.HARVESTMON_TENDERMINT_SERVICE_NAME,
		"MESSAGE":       alert.Message,
	}

	for k, v := range alert.Alarmer.AlarmParamList {
		payload[k] = applyReplaceIfString(v, alarmMap)
	}
	payload["text"] = alert.Message

	// Send the alert using the client's Lambda function
	client.InvokeLambda(alert.Alarmer.FunctionName, payload, true)

	// Generate a new UUID for the alert record
	alertRecordUUID, err := uuid.NewUUID()
	if err != nil {
		return fmt.Errorf("failed to generate UUID: %w", err)
	}

	// Save the alert record
	err = alertRecordRepository.Save(repository.AlertRecord{
		AlertRecordUUID: alertRecordUUID.String(),
		CreatedAt:       now,
		AlertName:       alert.AlertLevel.AlertName.String(),
		LevelName:       alert.AlertLevel.AlertLevel,
		AlarmerName:     alert.Alarmer.AlarmerName,
		AgentName:       string(alert.Agent),
		CommitID:        cfg.CommitId,
	})
	if err != nil {
		return fmt.Errorf("failed to save alert record: %w", err)
	}

	return nil
}

// Apply replacements if the value is a string, or recursively process slices/maps
func applyReplaceIfString(v any, definedWords map[string]string) any {
	val := reflect.ValueOf(v)
	switch val.Kind() {
	case reflect.String:
		return replaceDefinedWords(v.(string), definedWords)
	case reflect.Slice:
		newSlice := reflect.MakeSlice(val.Type(), val.Len(), val.Cap())
		for i := 0; i < val.Len(); i++ {
			newSlice.Index(i).Set(reflect.ValueOf(applyReplaceIfString(val.Index(i).Interface(), definedWords)))
		}
		return newSlice.Interface()
	case reflect.Map:
		newMap := reflect.MakeMap(val.Type())
		for _, key := range val.MapKeys() {
			newMap.SetMapIndex(key, reflect.ValueOf(applyReplaceIfString(val.MapIndex(key).Interface(), definedWords)))
		}
		return newMap.Interface()
	default:
		return v
	}
}

// Replace placeholders in the string with defined values
func replaceDefinedWords(input string, definedWords map[string]string) string {
	re := regexp.MustCompile(`\$(\w+)`)
	return re.ReplaceAllStringFunc(input, func(matched string) string {
		key := strings.TrimPrefix(matched, "$")
		if val, exists := definedWords[key]; exists {
			return val
		}
		return matched
	})
}

func aprintf(msg string, args ...any) string {
	return fmt.Sprintf(msg, args...)
}
