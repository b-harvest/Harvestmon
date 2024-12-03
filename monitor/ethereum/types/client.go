package types

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	database "github.com/b-harvest/Harvestmon/database"
	log "github.com/b-harvest/Harvestmon/log"
	gorm_mysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"net/http"
	"strconv"
	"time"
)

type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type MonitorClient struct {
	httpClient   HttpClient
	hostWithPort string
	timeout      time.Duration
	retries      int
	DB           *sql.DB
}

func NewMonitorClient(cfg *MonitorConfig, httpClient HttpClient, configFilePath string) *MonitorClient {
	hostWithPort := fmt.Sprintf("http://%s:%s", cfg.Agent.Host, strconv.Itoa(cfg.Agent.Port))

	db, err := database.GetDatabase(configFilePath, "")
	if err != nil {
		log.Fatal(err)
	}

	rpcClient := MonitorClient{
		httpClient:   httpClient,
		hostWithPort: hostWithPort,
		timeout:      *cfg.Agent.Timeout,
		retries:      3,
		DB:           db,
	}
	log.Info(fmt.Sprintf("monitoring host with port: %s", hostWithPort))
	return &rpcClient
}

func (r *MonitorClient) GetDatabase(batchSize int) *gorm.DB {
	if batchSize == 0 {
		batchSize = 100
	}
	gormDB, err := gorm.Open(gorm_mysql.New(gorm_mysql.Config{Conn: r.DB}), &gorm.Config{CreateBatchSize: batchSize, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		panic(err)
	}
	return gormDB
}

func (cfg *MonitorClient) GetBlockNumber() (int64, error) {
	if cfg == nil {
		return 0, errors.New("MonitorClient is nil in GetBlockNumber")
	}

	response, err := cfg.rpcCall("eth_blockNumber", []interface{}{})
	if err != nil {
		return 0, err
	}

	var result string
	if err = json.Unmarshal(response.Result, &result); err != nil {
		return 0, err
	}
	if result == "" {
		return 0, errors.New("Empty result from response")
	}

	var blockNumber int64
	_, err = fmt.Sscanf(result, "0x%x", &blockNumber)
	return blockNumber, err
}

func (cfg *MonitorClient) IsSyncing() (bool, error) {
	if cfg == nil {
		return false, errors.New("MonitorClient instance is nil")
	}

	response, err := cfg.rpcCall("eth_syncing", []interface{}{})
	if err != nil {
		return false, err
	}

	if response == nil {
		return false, errors.New("Nil response from rpcCall")
	}

	var syncingResult interface{}
	if err = json.Unmarshal(response.Result, &syncingResult); err != nil {
		return false, err
	}

	switch result := syncingResult.(type) {
	case bool:
		return result, nil
	case map[string]interface{}:
		return true, nil
	default:
		return false, errors.New("Unexpected result type for eth_syncing")
	}
}

func (cfg *MonitorClient) rpcCall(method string, params []interface{}) (*RPCResponse, error) {
	if cfg == nil {
		return nil, fmt.Errorf("MonitorClient instance is nil")
	}

	reqBody, err := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
		"id":      1,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", cfg.hostWithPort, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := cfg.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HTTP response status: %s", resp.Status)
	}

	var rpcResp RPCResponse
	if err = json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return nil, err
	}
	if rpcResp.Error != nil {
		return nil, errors.New(rpcResp.Error.Message)
	}

	return &rpcResp, nil
}
