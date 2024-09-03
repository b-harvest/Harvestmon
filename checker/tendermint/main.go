package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/b-harvest/Harvestmon/checker/tendermint/checker"
	"github.com/b-harvest/Harvestmon/checker/tendermint/types"
	_const "github.com/b-harvest/Harvestmon/const"
	"github.com/b-harvest/Harvestmon/log"
	"github.com/rs/zerolog"
	"gopkg.in/yaml.v3"
	"net/http"
	"os"
	"path/filepath"
	"sync"
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
	configBytes, err = os.ReadFile(filepath.Join(pwd, "resources/default_checker_rules.yaml"))
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

	// Parse default_alert_definition.yaml
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

func handler(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	_, err = http.NewRequest(event.HTTPMethod, event.Path, bytes.NewReader([]byte(event.Body)))
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
	}

	var header = make(map[string]string)
	for key, value := range event.Headers {
		header[key] = value
	}

	handleAction()

	log.Debug("Complete handling.... ")

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       "",
		Headers:    header,
	}, nil
}

func main() {
	lambda.Start(handler)
}

var DefaultCheckerRegistry = map[string]types.Func{
	"hearbeat":     checker.HeartbeatChecker,
	"block_commit": checker.BlockCommitChecker,
	"height_stuck": checker.HeightStuckChecker,
	"net_info":     checker.NetInfoChecker,
}

func handleAction() {

	customAgentConfigs := types.GetCustomAgentFiles()
	cfg.MergeWithCustomAgentChecker(customAgentConfigs)

	log.Info("Starting... Checker: " + _const.HARVESTMON_TENDERMINT_SERVICE_NAME + ", CommitID: " + cfg.CommitId)

	client, err = types.NewCheckerClient(&cfg, &alertDefinition, customAgentConfigs)
	if err != nil {
		log.Error(err)
	}

	var (
		wg sync.WaitGroup
	)

	for _, check := range types.ParseCheckerFunctions(DefaultCheckerRegistry) {
		wg.Add(1)
		go func(checker types.Checker) {
			defer wg.Done()
			checker.Run(&cfg, client)
		}(check)
	}
	wg.Wait()

	return
}
