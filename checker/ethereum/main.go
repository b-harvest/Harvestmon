package main

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/b-harvest/Harvestmon/checker/ethereum/checker"
	"github.com/b-harvest/Harvestmon/checker/ethereum/types"
	_const "github.com/b-harvest/Harvestmon/const"
	database "github.com/b-harvest/Harvestmon/database"
	"github.com/b-harvest/Harvestmon/log"
	"github.com/rs/zerolog"
	"gopkg.in/yaml.v3"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

var (
	cfg             = types.CheckerConfig{}
	alertDefinition = types.AlertDefinition{}
	wdb             *sql.DB
	rdb             *sql.DB
)

func init() {
	var (
		configBytes []byte
	)

	pwd, err := os.Getwd()
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

	flag.Parse()

	if *logLevelDebug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	wdb, err = database.GetDatabase("resources/default_checker_rules.yaml", "")
	rdb, err = database.GetDatabase("resources/default_checker_rules.yaml", "READ_")
	if err != nil {
		rdb = wdb
	}

}

func handler(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	_, err := http.NewRequest(event.HTTPMethod, event.Path, bytes.NewReader([]byte(event.Body)))
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
	defer func() {
		wdb.Close()
		rdb.Close()
	}()
	lambda.Start(handler)
}

var DefaultCheckerRegistry = map[string]types.Func{
	"hearbeat":     checker.HeartbeatChecker,
	"block_number": checker.BlockNumberChecker,
}

func handleAction() {

	customAgentConfigs := types.GetCustomAgentFiles()
	cfg.MergeWithCustomAgentChecker(customAgentConfigs)

	log.Info("Starting... Checker: " + _const.HARVESTMON_ETHEREUM_SERVICE_NAME + ", CommitID: " + cfg.CommitId)

	for _, al := range alertDefinition.AlertLevel {
		log.Debug(fmt.Sprintf("alert defined: %s:%s", al.AlertName, al.AlertLevel))
	}

	client, err := types.NewCheckerClient(&cfg, &alertDefinition, customAgentConfigs, wdb, rdb)
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

}
