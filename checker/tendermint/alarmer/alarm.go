package alarmer

import (
	"fmt"
	_const "github.com/b-harvest/Harvestmon/const"
	"github.com/b-harvest/Harvestmon/log"
	"github.com/b-harvest/Harvestmon/repository"
	"github.com/google/uuid"
	"reflect"
	"regexp"
	"strings"
	"tendermint-checker/types"
	"time"
)

func RunAlarm(cfg *types.CheckerConfig, client types.CheckerClient, alert types.Alert) error {
	alertRecordRepository := repository.AlertRecordRepository{BaseRepository: repository.BaseRepository{DB: *client.GetDatabase(), CommitId: cfg.CommitId}}

	now := time.Now().UTC()
	startTime := now.Add(-(*alert.Alarmer.AlarmResendDuration))

	result, err := alertRecordRepository.ExistsIfAlertRecordIsMarkedOrAlreadySent(
		alert.AlertLevel.AlertName.String(),
		alert.Alarmer.AlarmerName,
		string(alert.Agent),
		startTime, now, 30*time.Minute)
	if err != nil {
		return err
	}

	if result {
		log.Info(aprintf("Alert has already sent to target within %v or Marked by operator. agent: %s, alert: %s", alert.Alarmer.AlarmResendDuration, alert.Agent, alert.AlertLevel.AlertName))
		return nil
	}

	log.Info(aprintf(strings.Replace(alert.Message, "\n", ". ", -1)))

	alertRecordUUID, err := uuid.NewUUID()
	if err != nil {
		return err
	}

	err = alertRecordRepository.Save(
		repository.AlertRecord{
			AlertRecordUUID: alertRecordUUID.String(),
			CreatedAt:       time.Now().UTC(),
			AlertName:       alert.AlertLevel.AlertName.String(),
			LevelName:       alert.AlertLevel.AlertLevel,
			AlarmerName:     alert.Alarmer.AlarmerName,
		})

	if err != nil {
		return err
	}

	var (
		payload  = make(map[string]any)
		alarmMap = map[string]string{
			"AGENT":         string(alert.Agent),
			"ALERT_NAME":    string(alert.AlertLevel.AlertName),
			"ALERT_LEVEL":   alert.AlertLevel.AlertLevel,
			"ALERT_SERVICE": _const.HARVESTMON_TENDERMINT_SERVICE_NAME,
			"MESSAGE":       alert.Message,
		}
	)

	for k, v := range alert.Alarmer.AlarmParamList {
		payload[k] = applyReplaceIfString(v, alarmMap)
	}
	payload["text"] = alert.Message
	client.InvokeLambda(alert.Alarmer.AlarmerName, payload, true)
	return nil
}

func applyReplaceIfString(v any, definedWords map[string]string) any {
	switch reflect.TypeOf(v).Kind() {
	case reflect.String:
		// v is a string, apply the replacement function
		return replaceDefinedWords(v.(string), definedWords)
	case reflect.Slice:
		// v is a slice, apply the function to each element
		s := reflect.ValueOf(v)
		for i := 0; i < s.Len(); i++ {
			s.Index(i).Set(reflect.ValueOf(applyReplaceIfString(s.Index(i).Interface(), definedWords)))
		}
	case reflect.Map:
		// v is a map, apply the function to each value
		m := reflect.ValueOf(v)
		for _, key := range m.MapKeys() {
			m.SetMapIndex(key, reflect.ValueOf(applyReplaceIfString(m.MapIndex(key).Interface(), definedWords)))
		}
	}
	return v
}

func replaceDefinedWords(input string, definedWords map[string]string) string {
	// Regular expression to find words starting with $
	re := regexp.MustCompile(`\$(\w+)`)

	// Function to replace matched word
	result := re.ReplaceAllStringFunc(input, func(matched string) string {
		// Remove the $ and check if the word exists in the map
		key := strings.TrimPrefix(matched, "$")
		if val, exists := definedWords[key]; exists {
			// Replace the entire $word with the map value
			return val
		}
		// If not found in the map, return the original matched string
		return matched
	})

	return result
}

func aprintf(msg string, args ...any) string {
	return fmt.Sprintf("[alert] "+msg, args...)
}
