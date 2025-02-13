package main

import (
	"encoding/json"
	"errors"
	"flag"
	"github.com/aws/aws-lambda-go/events"
	"github.com/prometheus/common/model"
	log "github.com/sirupsen/logrus"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	alertManager *AlertManager
)

func init() {
	time.Local = time.UTC

	var (
		logLevel   string
		configFile string
		l          log.Level
		err        error
	)

	err = InitializeViper()
	if err != nil {
		panic(err)
	}

	flag.StringVar(&logLevel, "log-level", "", "allow showing debug log")
	flag.StringVar(&configFile, "config", "", "configuration file path")
	flag.Parse()
	if logLevel == "" && os.Getenv("LOG_LEVEL") != "" {
		logLevel = os.Getenv("LOG_LEVEL")
	} else {
		logLevel = "info"
	}

	if configFile == "" {
		if os.Getenv("CONFIG_PATH") != "" {
			configFile = os.Getenv("CONFIG_PATH")
		} else {
			configFile = "./config.toml"
		}
	}

	log.AddHook(&ErrorHook{})
	l, err = log.ParseLevel(logLevel)
	if err != nil {
		panic(err)
	}
	log.SetLevel(l)

	alertManager = NewAlertManager(log.NewEntry(log.StandardLogger()))
	if err = alertManager.LoadConfig(configFile); err != nil {
		log.Fatalf("Error loading configuration: %v", err)
		panic(err)
	}

	alertManager.GithubFile.isChanged()

}

func main() {
	start()
}

func handleAction(snsEvent *events.SNSEvent) error {
	err := alertManager.loadAlerts()
	if err != nil {
		log.Fatalf("Error loading alerts: %v", err)
		panic(err)
	}

	defer func() {
		err = alertManager.saveAlerts(time.Now())
		if err != nil {
			panic(err)
		}
	}()

	if err = alertManager.getConfig(); err != nil {
		return err
	}

	var (
		wg       sync.WaitGroup
		alertMap = make(map[string]AlertMessage)
	)

	for _, record := range snsEvent.Records {
		log.Debugf("SNS record %v", record.SNS.Message)
		rawAlrtMsg := record.SNS.Message
		var alrtMsgs AlertMessages

		rawAlrtMsg = strings.ReplaceAll(rawAlrtMsg, "\\n", "\n")
		rawAlrtMsg = strings.ReplaceAll(rawAlrtMsg, "\n", "\\n")

		err = json.Unmarshal([]byte(rawAlrtMsg), &alrtMsgs)
		if err != nil {
			alertManager.logger.Errorf("failed to unmarshal alarm message: %v, body: %s", err, rawAlrtMsg)
			continue
		}

		for _, alrtMsg := range alrtMsgs.Alerts {

			labels := make(map[string]string)
			for k, v := range alrtMsg.Labels {
				if k == "" || v == "" {
					continue
				}
				labels[k] = v
			}
			alrtMsg.Labels = labels
			if _, ok := alertMap[string(alrtMsg.Instance)+alrtMsg.Summary]; ok && alrtMsg.Status == model.AlertResolved {
			} else {
				alertMap[string(alrtMsg.Instance)+alrtMsg.Summary] = alrtMsg
			}
		}
	}

	for _, alrtMsg := range alertMap {
		wg.Add(1)
		go func() {
			defer wg.Done()
			alertManager.alert(alrtMsg)
		}()
	}

	wg.Wait()
	alertManager.logger.Info("processing alert finished")
	return nil
}

type ErrorHook struct {
	Fired bool
}

func (hook *ErrorHook) Fire(entry *log.Entry) error {
	hook.Fired = true

	var (
		alertLevel = "alertmanager"
		alarmers   []Alarmer
	)
	if alertManager == nil || alertManager.AlarmerConfig == nil {
		return errors.New("alert manager config is nil")
	}

	for _, almr := range alertManager.AlarmerConfig.getAlarmers() {
		for _, tl := range almr.getTargetAlertLevels() {
			if tl.String() == alertLevel {
				alarmers = append(alarmers, almr)
				break
			}
		}
	}

	var errs []error
	for _, alarmer := range alarmers {
		line, err := entry.Bytes()
		if err != nil {
			errs = append(errs, err)
			continue
		}

		_, err = alarmer.notify(&AlertMessage{
			Instance:   "alertmanager",
			Target:     "",
			StartsAt:   time.Now().String(),
			Summary:    string(line),
			Status:     model.AlertFiring,
			AlertEvent: "alertmanager:error",
			AlertLevel: AlertLevel(alertLevel),
		})
		if err != nil {
			errs = append(errs, err)
		}
	}
	if errs != nil {
		return errs[0]
	}

	return nil
}

func (hook *ErrorHook) Levels() []log.Level {
	return []log.Level{
		log.ErrorLevel,
	}
}
