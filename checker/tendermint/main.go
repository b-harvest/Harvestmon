package main

import (
	"errors"
	"github.com/b-harvest/Harvestmon/log"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"sync"
	"tendermint-checker/checker"
	"tendermint-checker/types"
	monotir_types "tendermint-mon/types"
	"time"
)

var (
	err    error
	client *types.CheckerClient
	cfg    = types.CheckerConfig{}
)

func init() {
	var configBytes []byte

	pwd, err := os.Getwd()
	configBytes, err = os.ReadFile(filepath.Join(pwd, "resources/config.yaml"))
	if err != nil {
		log.Fatal(err)
	}

	err = yaml.Unmarshal(configBytes, &cfg)
	if err != nil {
		log.Fatal(err)
	}

	err = cfg.ApplyConfigFromEnvAndDefault()
	if err != nil {
		log.Fatal(errors.New("Error occurred while parsing env. " + err.Error()))
	}

}

var CheckerRegistry = map[string]types.Func{
	"hearbeat":     checker.HeartbeatChecker,
	"block_commit": checker.BlockCommitChecker,
	"status":       checker.HeightStuckChecker,
	"net_info":     checker.NetInfoChecker,
}

func main() {
	log.Info("Starting... Checker: " + monotir_types.HARVEST_SERVICE_NAME + ", CommitID: " + cfg.CommitId)
	client = types.NewCheckerClient(&cfg)

	var (
		wg sync.WaitGroup
	)

	ticker := time.NewTicker(cfg.CheckInterval)
	done := make(chan bool)
	for _, mon := range CheckerRegistry {
		wg.Add(1)
		go func(monitor types.Checker) {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				case <-ticker.C:
					monitor.Run(&cfg, client)
				}
			}
		}(mon)
	}
	wg.Wait()
	ticker.Stop()

	return
}
