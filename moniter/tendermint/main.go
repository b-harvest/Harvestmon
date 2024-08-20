package main

import (
	"errors"
	"flag"
	_const "github.com/b-harvest/Harvestmon/const"
	log "github.com/b-harvest/Harvestmon/log"
	"github.com/rs/zerolog"
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
	client  *types.MonitorClient
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

	logLevelDebug := flag.Bool("debug", false, "allow showing debug log")

	flag.Parse()

	if *logLevelDebug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

}

func main() {
	log.Info("Starting... Agent: " + mConfig.Agent.AgentName + ", Service: " + _const.HARVESTMON_TENDERMINT_SERVICE_NAME + ", CommitId: " + mConfig.Agent.CommitId)

	client = types.NewMonitorClient(&mConfig, &http.Client{Timeout: *mConfig.Agent.Timeout})

	var (
		wg   sync.WaitGroup
		svcs = mConfig.Agent.Monitors
	)

	ticker := time.NewTicker(*mConfig.Agent.PushInterval)
	done := make(chan bool)
	for _, mon := range svcs {
		wg.Add(1)
		go func(monitor types.Monitor) {
			monitor.Run(&mConfig, client)
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				case <-ticker.C:
					monitor.Run(&mConfig, client)
				}
			}
		}(mon)
	}
	wg.Wait()
	ticker.Stop()

	return
}
