package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	_const "github.com/b-harvest/Harvestmon/const"
	"github.com/b-harvest/Harvestmon/repository"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
)

type EthereumMonitorConfig struct {
	logger *log.Entry

	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`

	agentName string
	commitId  string

	monitorTarget MonitorTarget
	Collectors    []Collector `mapstructure:"collectors"`

	client *http.Client

	batchSize int
}

func (c *EthereumMonitorConfig) Validate() error {
	if c.logger == nil {
		return errors.New("logger is required")
	}

	if c.Host == "" {
		return errors.New("host is required")
	}

	if c.Port == 0 {
		return errors.New("port is required")
	}

	if len(c.Collectors) == 0 {
		return errors.New("collectors is required")
	}

	if c.agentName == "" {
		return errors.New("agent name is required")
	}

	if c.commitId == "" {
		return errors.New("commitId is required")
	}

	if c.batchSize == 0 {
		return errors.New("batchSize is required")
	}

	if c.client == nil {
		return errors.New("client is required")
	}

	for _, collector := range c.Collectors {
		if err := collector.Validate(); err != nil {
			return errors.Wrapf(err, "collector '%v' is invalid", collector.Name)
		}
	}
	return nil
}

func (c *EthereumMonitorConfig) initialize(
	agentName, commitId string,
	batchSize int,
	logger *log.Entry,
	client *http.Client,
	collectors []Collector) {

	c.agentName = agentName
	c.commitId = commitId
	c.batchSize = batchSize
	c.logger = logger
	c.client = client
	c.Collectors = collectors
}

func (c *EthereumMonitorConfig) getMonitorName() string {
	return _const.HARVESTMON_ETHEREUM_SERVICE_NAME
}

func (c *EthereumMonitorConfig) getLogger() *log.Entry {
	return c.logger
}

func (c *EthereumMonitorConfig) getCollectors() []Collector {
	return c.Collectors
}

func (c *EthereumMonitorConfig) load(r *repository.BaseRepository) error {
	return nil
}

func (c *EthereumMonitorConfig) exit(repo *repository.BaseRepository) error {
	return nil
}

var ethBlockNumberCollector = func(mc MonitorConfig) func(sq chan StoreEntity, l *log.Entry) {
	c, ok := mc.(*EthereumMonitorConfig)
	if !ok {
		return func(sq chan StoreEntity, l *log.Entry) {
			l.Warningf("invalid config type")
			return
		}
	}

	return func(sq chan StoreEntity, l *log.Entry) {
		endpoint := getEndpoint(c.Host, c.Port)

		ctx, cancel := context.WithTimeout(context.Background(), c.client.Timeout)
		defer cancel()

		rpcRes, err := rpcCall(c.client, ctx, endpoint, "eth_blockNumber", []interface{}{})
		if err != nil {
			c.logger.Warningf("failed to call eth_blockNumber: %v", err)
			return
		}

		var result string
		if err = json.Unmarshal(rpcRes.Result, &result); err != nil {
			c.logger.Warningf("failed to unmarshal eth_blockNumber: %v", err)
			return
		}
		if result == "" {
			c.logger.Warningf("result is empty")
			return
		}

		var blockNumber int64
		_, err = fmt.Sscanf(result, "0x%x", &blockNumber)
		if err != nil {
			c.logger.Warningf("failed to scan eth_blockNumber(%v): %v", result, err)
			return
		}

		ethBlockNumber, err := newEthBlockNumber(c.agentName, c.commitId, blockNumber)
		if err != nil {
			l.Error(errors.Wrapf(err, "failed to parse ethBlockNumber response '%s'", endpoint).Error())
			return
		}

		c.logger.Debugf("eth_blockNumber %v", ethBlockNumber.BlockNumber)
		sq <- StoreEntity{
			ethBlock: ethBlockNumber,
		}

		return
	}
}

func newEthBlockNumber(agentName, commitId string, number int64) (*repository.EthereumBlockNumber, error) {
	eventUUID, err := uuid.NewUUID()
	if err != nil {
		log.Error(err)
	}
	createdAt := time.Now().UTC()

	return &repository.EthereumBlockNumber{
		CreatedAt: createdAt,
		EventUUID: eventUUID.String(),
		Event: repository.Event{
			EventUUID:   eventUUID.String(),
			AgentName:   agentName,
			ServiceName: _const.HARVESTMON_ETHEREUM_SERVICE_NAME,
			CommitID:    commitId,
			EventType:   _const.ETH_BLOCK_NUMBER_EVENT_TYPE,
			CreatedAt:   createdAt,
		},
		BlockNumber: fmt.Sprintf("%d", number),
	}, nil
}

func rpcCall(client *http.Client, ctx context.Context, endpoint string, method string, params []interface{}) (*RPCResponse, error) {
	reqBody, err := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
		"id":      1,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal json")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build reqest '%s'", endpoint)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to send reqest '%s'", endpoint)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(fmt.Sprintf("failed to send reqest: %s, status: %v", endpoint, resp.StatusCode))
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
