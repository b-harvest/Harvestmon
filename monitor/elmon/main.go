package main

import (
	"errors"
	"flag"
	"fmt"
	_const "github.com/b-harvest/Harvestmon/const"
	log "github.com/b-harvest/Harvestmon/log"
	"github.com/b-harvest/Harvestmon/monitor/elmon/monitor"
	"github.com/b-harvest/Harvestmon/monitor/elmon/types"
	"github.com/rs/zerolog"
	"gopkg.in/yaml.v3"
	"net/http"
	"os"
	logs "log"
	"path/filepath"
	"sync"
	"time"
)

var (
	err     error
	client  *types.MonitorClient
	mConfig = types.MonitorConfig{}
)

func init() {
	logs.Printf("MonitorConfig initialized: %+v\n", mConfig)
	// initializing monitor functions
	types.MonitorRegistry = map[string]types.Func{
		"execution": {MonitorFunc: monitor.ExecutionMonitor},
	}

	var configBytes []byte

	configFilePath := os.Getenv(types.EnvConfigFilePath)
	if configFilePath == "" {
		configFilePath = "resources/config.yaml"
	}

	if !filepath.IsAbs(configFilePath) {
		pwd, _ := os.Getwd()
		configFilePath = filepath.Join(pwd, configFilePath)
	}

	configBytes, err = os.ReadFile(configFilePath)
	if err != nil {
		log.Fatal(err)
	}

	err = yaml.Unmarshal(configBytes, &mConfig)
	if err != nil {
		log.Fatal(err)
	}

	err = mConfig.ApplyConfigFromEnvAndDefault()
	if err != nil {
		log.Fatal(errors.New("Error occurred while parsing env. " + err.Error()))
	}

	logs.Printf("MonitorConfig after loading from file: %+v\n", mConfig)


	client, err = types.NewMonitorClient(&mConfig, &http.Client{Timeout: *mConfig.Agent.Timeout}, configFilePath)
	if err != nil {
        log.Fatal(errors.New("Failed to create monitor client: " + err.Error()))
    }
    
    if client == nil {
        log.Fatal(errors.New("Monitor client is nil after initialization"))
    } else {
        log.Info("Monitor client initialized successfully")
    }


	logLevelDebug := flag.Bool("debug", false, "allow showing debug log")

	flag.Parse()

	if *logLevelDebug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

}

func main() {
	if client == nil {
        log.Fatal(errors.New("Monitor client is not properly initialized"))
        return
    }
	// defer client.DB.Close()
	log.Info("Starting... Agent: " + mConfig.Agent.AgentName + ", Service: " + _const.HARVESTMON_TENDERMINT_SERVICE_NAME + ", CommitId: " + mConfig.Agent.CommitId)

	var (
		wg   sync.WaitGroup
		svcs = mConfig.Agent.Monitors
	)

	done := make(chan bool)
	for _, mon := range svcs {
		wg.Add(1)

		loopMon := mon
		go func(monitor types.Monitor) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					log.Error(errors.New(fmt.Sprintf("Panic recovered in monitor: %v", r)))
					time.Sleep(*loopMon.Interval) // Sleep to prevent tight loop
				}
			}()

			ticker := time.NewTicker(*mConfig.Agent.PushInterval)

			for {
				select {
				case <-done:
					ticker.Stop() // Gracefully stop the ticker
					return
				case <-ticker.C:
					if client == nil {
						log.Fatal(errors.New("monitor.run error"))
					}
					logs.Printf("Running monitor: %+v with client: %+v\n", monitor, client) // 디버깅을 위한 추가 로깅

					monitor.Run(&mConfig, client)
				}
			}
		}(mon)
	}

	wg.Wait()

	return
}
