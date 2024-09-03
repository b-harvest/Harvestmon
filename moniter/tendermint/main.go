package main

import (
	"errors"
	"flag"
	_const "github.com/b-harvest/Harvestmon/const"
	log "github.com/b-harvest/Harvestmon/log"
	"github.com/b-harvest/Harvestmon/moniter/tendermint/monitor"
	"github.com/b-harvest/Harvestmon/moniter/tendermint/types"
	"github.com/rs/zerolog"
	"gopkg.in/yaml.v3"
	"net/http"
	"os"
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
	types.MonitorRegistry = map[string]types.Func{
		"net_info":     {monitor.NetInfoMonitor, nil},
		"block_commit": {monitor.BlockCommitMonitor, nil},
		"status":       {monitor.CometBFTStatusMonitor, nil},
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

	client = types.NewMonitorClient(&mConfig, &http.Client{Timeout: *mConfig.Agent.Timeout}, configFilePath)

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

	var (
		wg   sync.WaitGroup
		svcs = mConfig.Agent.Monitors
	)

	ticker := time.NewTicker(*mConfig.Agent.PushInterval)
	done := make(chan bool)
	for _, mon := range svcs {
		wg.Add(1)
		if mon.Interval != nil && *mon.Interval > 0 {
			ticker = time.NewTicker(*mon.Interval)
		}
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
