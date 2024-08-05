package main

import (
	"errors"
	log "github.com/b-harvest/harvestmon-log"
	"gopkg.in/yaml.v3"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"tendermint-mon/monitor"
	"tendermint-mon/types"
	"time"
)

var (
	err     error
	tClient *types.MonitorClient
	mConfig = types.MonitorConfig{}
)

func init() {
	types.MonitorRegistry = map[string]types.Func{
		"net_info":     monitor.NetInfoMonitor,
		"block_commit": monitor.BlockCommitMonitor,
		"status":       monitor.CometBFTStatusMonitor,
	}

	var configBytes []byte

	pwd, err := os.Getwd()
	configBytes, err = os.ReadFile(filepath.Join(pwd, "resources/config.yaml"))
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

}

func main() {
	log.Info("Starting... Agent: " + mConfig.Agent.AgentName + ", Service: " + types.HARVEST_SERVICE_NAME + ", CommitID: " + mConfig.Agent.CommitId)
	tClient = types.NewMonitorClient(&mConfig, &http.Client{Timeout: mConfig.Agent.Timeout})

	var (
		wg   sync.WaitGroup
		svcs = mConfig.Agent.Monitors
	)

	ticker := time.NewTicker(mConfig.Agent.PushInterval)
	done := make(chan bool)
	for _, mon := range svcs {
		wg.Add(1)
		go func(monitor types.Monitor) {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				case <-ticker.C:
					monitor.Run(&mConfig, tClient)
				}
			}
		}(mon)
	}
	wg.Wait()
	ticker.Stop()

	return
}
