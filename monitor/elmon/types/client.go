package types

import (
	// "context"
	// "database/sql"
	"encoding/json"
	"errors"
	"fmt"
	// database "github.com/b-harvest/Harvestmon/database"
	// log "github.com/b-harvest/Harvestmon/log"
	// gorm_mysql "gorm.io/driver/mysql"
	// "gorm.io/gorm"
	// "gorm.io/gorm/logger"
	// "io"
	"net/http"
	// "reflect"
	// "runtime"
	"strconv"
	// "strings"
    "log"
	"time"
	"bytes"
)

type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// MonitorClient 구조체 정의
type MonitorClient struct {
	httpClient   HttpClient
	hostWithPort string
	timeout      time.Duration
	retries      int
	// DB           *sql.DB
}


func NewMonitorClient(cfg *MonitorConfig, httpClient HttpClient, configFilePath string) (*MonitorClient, error) {
    if cfg == nil || httpClient == nil {
        return nil, fmt.Errorf("configuration or httpClient cannot be nil")
    }
	hostWithPort := fmt.Sprintf("http://%s:%s", cfg.Agent.Host, strconv.Itoa(cfg.Agent.Port))

	// db, err := database.GetDatabase(configFilePath)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	rpcClient := MonitorClient{
		httpClient:   httpClient,
		hostWithPort: hostWithPort,
		timeout:      *cfg.Agent.Timeout,
		retries:      3,
		// DB:           db,
	}
    log.Printf("MonitorClient initialized: %+v\n", rpcClient)
	return &rpcClient, nil
}

// func (r *MonitorClient) GetDatabase(batchSize int) *gorm.DB {
// 	if batchSize == 0 {
// 		batchSize = 100
// 	}
// 	gormDB, err := gorm.Open(gorm_mysql.New(gorm_mysql.Config{Conn: r.DB}), &gorm.Config{CreateBatchSize: batchSize, Logger: logger.Default.LogMode(logger.Silent)})
// 	if err != nil {
// 		panic(err)
// 	}
// 	return gormDB
// }

// eth_blockNumber를 요청하는 메소드
func (cfg *MonitorClient) GetBlockNumber() (int64, error) {
    if cfg == nil {
        fmt.Println("GetBlockNumber called with a nil MonitorClient")
		return 0, errors.New("MonitorClient is nil in GetBlockNumber")
    }

    log.Println("Calling rpcCall for eth_blockNumber...")
    response, err := cfg.rpcCall("eth_blockNumber", []interface{}{})
    if err != nil {
        fmt.Println("Error in rpcCall:", err)
        return 0, err
    }

    log.Printf("RPC Response: %+v\n", response)
    var result string
    if err := json.Unmarshal(response.Result, &result); err != nil {
        fmt.Println("Error unmarshaling response result:", err)
        return 0, err
    }
    if result == "" {
        log.Println("Received empty result from response")
        return 0, errors.New("Empty result from response")
    }

    var blockNumber int64
    _, err = fmt.Sscanf(result, "0x%x", &blockNumber)
    if err != nil {
        fmt.Println("Error parsing block number:", err)
    } else {
        fmt.Println("Parsed block number:", blockNumber)
    }
    return blockNumber, err
}

// eth_syncing를 요청하는 메소드
func (cfg *MonitorClient) IsSyncing() (bool, error) {
    if cfg == nil {
        log.Println("IsSyncing called with a nil MonitorClient")
        return false, fmt.Errorf("MonitorClient instance is nil")
    }

    log.Println("Calling rpcCall for eth_syncing...")
    response, err := cfg.rpcCall("eth_syncing", []interface{}{})
    if err != nil {
        log.Printf("Error in rpcCall: %v\n", err)
        return false, err
    }

    if response == nil {
        log.Println("Received nil response from rpcCall")
        return false, errors.New("Nil response from rpcCall")
    }

    log.Printf("RPC Response: %+v\n", response)

    var syncingResult interface{}
    if err := json.Unmarshal(response.Result, &syncingResult); err != nil {
        log.Printf("Error unmarshaling response result: %v\n", err)
        return false, err
    }

    switch result := syncingResult.(type) {
    case bool:
        log.Printf("Node syncing status: %v\n", result)
        return result, nil
    case map[string]interface{}:
        log.Println("Node is syncing")
        return true, nil
    default:
        log.Println("Unexpected result type for eth_syncing")
        return false, errors.New("Unexpected result type for eth_syncing")
    }
}


// JSON RPC 호출을 처리하는 함수
func (cfg *MonitorClient) rpcCall(method string, params []interface{}) (*RPCResponse, error) {
    if cfg == nil {
        log.Println("rpcCall called with a nil MonitorClient")
        return nil, fmt.Errorf("MonitorClient instance is nil")
    }

    log.Printf("Preparing RPC call with method: %s, params: %+v\n", method, params)


    reqBody, err := json.Marshal(map[string]interface{}{
        "jsonrpc": "2.0",
        "method":  method,
        "params":  params,
        "id":      1,
    })
    if err != nil {
        log.Printf("Error marshaling request body: %v\n", err)
        return nil, err
    }


    log.Printf("Request body: %s\n", reqBody)


    req, err := http.NewRequest("POST", cfg.hostWithPort, bytes.NewBuffer(reqBody))
    if err != nil {
        log.Printf("Error creating new HTTP request: %v\n", err)
        return nil, err
    }
    req.Header.Set("Content-Type", "application/json")

    resp, err := cfg.httpClient.Do(req)
    if err != nil {
        log.Printf("Error making HTTP request: %v\n", err)
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        log.Printf("Unexpected HTTP response status: %s\n", resp.Status)
        return nil, fmt.Errorf("unexpected HTTP response status: %s", resp.Status)
    }

    var rpcResp RPCResponse
    if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
        log.Printf("Error decoding RPC response: %v\n", err)
        return nil, err
    }
    
    if rpcResp.Error != nil {
        log.Printf("RPC Error: Code %d, Message: %s\n", rpcResp.Error.Code, rpcResp.Error.Message)
    } else {
        log.Printf("RPC Response decoded successfully: %+v\n", rpcResp)
    }


    return &rpcResp, nil
}

type RPCResponse struct {
    Jsonrpc string          `json:"jsonrpc"`
    Result  json.RawMessage `json:"result"`
    Error   *RPCErr         `json:"error"`
    ID      int             `json:"id"`
}

type RPCErr struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
}
