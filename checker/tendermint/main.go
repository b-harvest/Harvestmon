package main

import (
	"errors"
	"flag"
	_const "github.com/b-harvest/Harvestmon/const"
	"github.com/b-harvest/Harvestmon/log"
	"github.com/rs/zerolog"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"tendermint-checker/types"
	"time"
)

var (
	err             error
	client          *types.CheckerClient
	cfg             = types.CheckerConfig{}
	alertDefinition = types.AlertDefinition{}
	agentFilesPath  *string
	pwd             string
)

func init() {
	var (
		configBytes []byte
	)

	pwd, err = os.Getwd()
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

	// Parse default_alert.yaml
	customDefinition, err := types.ParseAlertDefinition()
	if err != nil {
		log.Fatal(err)
	}

	alertDefinition = *customDefinition

	logLevelDebug := flag.Bool("debug", false, "allow showing debug log")
	agentFilesPath = flag.String("agent-files", "", "allow showing debug log")

	flag.Parse()

	if *logLevelDebug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	os.Getenv("z")

}

func main() {

	var customAgentConfigs []types.CustomAgentConfig

	var agentFiles []string
	if *agentFilesPath != "" {
		agentFiles = strings.Split(*agentFilesPath, ",")
	}

	for _, agentFile := range agentFiles {
		log.Debug("Parsing custom agent file... " + agentFile)

		var (
			agentFileContentBytes []byte
			customAgentFile       = types.CustomAgentConfig{}
			err                   error
		)
		if filepath.IsAbs(agentFile) {
			agentFileContentBytes, err = os.ReadFile(agentFile)
		} else {
			agentFileContentBytes, err = os.ReadFile(filepath.Join(pwd, agentFile))
		}

		if err != nil {
			log.Fatal(err)
		}

		err = yaml.Unmarshal(agentFileContentBytes, &customAgentFile)
		if err != nil {
			log.Fatal(err)
		}

		customAgentConfigs = append(customAgentConfigs, customAgentFile)
	}

	cfg.MergeWithCustomAgentChecker(customAgentConfigs)

	log.Info("Starting... Checker: " + _const.HARVESTMON_TENDERMINT_SERVICE_NAME + ", CommitID: " + cfg.CommitId)

	client, err = types.NewCheckerClient(&cfg, &alertDefinition, customAgentConfigs)
	if err != nil {
		log.Error(err)
	}

	var (
		wg sync.WaitGroup
	)

	ticker := time.NewTicker(*cfg.CheckInterval)
	done := make(chan bool)

	for _, mon := range types.ParseCheckerFunctions() {
		wg.Add(1)
		go func(monitor types.Checker) {
			monitor.Run(&cfg, client)
			defer wg.Done()
			for {
				select {
				case <-ticker.C:
					monitor.Run(&cfg, client)
				case <-done:
					return
				}
			}
		}(mon)
	}
	wg.Wait()
	ticker.Stop()

	return
}
