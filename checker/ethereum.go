package main

import (
	"encoding/json"
	"fmt"
	"github.com/b-harvest/Harvestmon/repository"
	"github.com/b-harvest/Harvestmon/util"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type EthereumHeightStrategy struct {
	logger *log.Entry

	Enabled      bool           `toml:"enabled"`
	MaxStuckTime *time.Duration `toml:"maxStuckTime"`
	AlertLevel   AlertLevel     `toml:"alertLevel"`

	HeartbeatTime *time.Duration `toml:"heartbeatTime"` // if not set, defaults to 3 minutes

	repo *repository.Repository
}

func (s *EthereumHeightStrategy) checkValid() error {
	if !s.Enabled {
		return nil
	}

	if s.MaxStuckTime == nil {
		return errors.New("maxStuckTime must be greater than zero")
	}
	if s.HeartbeatTime == nil {
		heatbeat := 3 * time.Minute
		s.HeartbeatTime = &heatbeat
	}

	return s.AlertLevel.checkValid()
}

func (e *EthereumHeightStrategy) getName() string {
	return "height"
}

func (s *EthereumHeightStrategy) getAlertLevel() AlertLevel {
	return s.AlertLevel
}

func (e *EthereumHeightStrategy) initialize(logger *log.Entry, repo *repository.Repository) {
	e.logger = logger
	e.repo = repo
}

func (s *EthereumHeightStrategy) check(nodeName string) (alertEvent, string, bool) {
	_, _, fn := util.TraceFirst()
	s.logger.Tracef(fmt.Sprintf("Check function: %s", fn))

	latestEthBlockNumbers, err := s.repo.FindEthereumBlockNumberByAgentNameWithLimit(nodeName, time.Now().Add(-*s.HeartbeatTime), 100)

	if err != nil {
		s.logger.Error(errors.Wrap(err, "Error finding latest eth block number").Error())
		return EthStuckAlarmEvent, "", true
	}

	var (
		initBlockNumber              = repository.EthereumBlockNumber{}
		currentBlockNumberTime       time.Time
		beforeChangedBlockNumberTime time.Time
	)

	for _, ethBlockNumber := range latestEthBlockNumbers {
		if initBlockNumber.BlockNumber == "" {
			initBlockNumber = ethBlockNumber
			currentBlockNumberTime = initBlockNumber.CreatedAt
			beforeChangedBlockNumberTime = ethBlockNumber.CreatedAt
		} else {
			beforeChangedBlockNumberTime = ethBlockNumber.CreatedAt
			if ethBlockNumber.BlockNumber == initBlockNumber.BlockNumber {
			} else {
				break
			}
		}
	}

	// no ethBlockNumbers detected -> Heartbeat error
	if len(latestEthBlockNumbers) == 0 {
		return EthHeartbeatAlarmEvent, fmt.Sprintf("Heartbeat check failed (%v)", s.HeartbeatTime), true
	}

	// height doesn't change for a while
	if (time.Time{}.Equal(beforeChangedBlockNumberTime)) || currentBlockNumberTime.UTC().Sub(beforeChangedBlockNumberTime) > *s.MaxStuckTime {

		return EthStuckAlarmEvent, fmt.Sprintf("EthBlockNumber has stuck for a while.\n\nblock: %s \ntime: %s UTC\nthreshold: %s",
			initBlockNumber.BlockNumber, beforeChangedBlockNumberTime.Format(time.DateTime), s.MaxStuckTime), true
	}

	return EthStuckAlarmEvent, fmt.Sprintf("Complete to check Agents:(%s) new block height: %s", nodeName, initBlockNumber.BlockNumber), false
}

type EthereumBehindStrategy struct {
	logger *log.Entry

	Enabled             bool       `toml:"enabled"`
	MaxBehindBlockCount uint       `toml:"maxBehindBlockCount"`
	TargetRPCs          []string   `toml:"targetRPCs"` // targetRPCs must be formatted with full syntax. e.g.) http(s)://<host>(:<port>)/jsonrpc
	AlertLevel          AlertLevel `toml:"alertLevel"`

	APITimeout time.Duration `toml:"apiTimeout"` // if not set, defaults to 10s

	HeartbeatTime *time.Duration `toml:"heartbeatTime"` // if not set, defaults to 3 minutes.

	rpc  JsonRPC
	repo *repository.Repository
}

func (s *EthereumBehindStrategy) checkValid() error {
	if !s.Enabled {
		return nil
	}

	if s.MaxBehindBlockCount == 0 {
		return errors.New("maxBehindBlockCount must be greater than zero")
	}

	for _, rpc := range s.TargetRPCs {
		if rpc == "" {
			return errors.New("target rpc must not be empty")
		}
		_, err := url.Parse(rpc)
		if err != nil {
			return errors.Wrapf(err, "Invalid target RPC: %s", rpc)
		}
	}

	if s.HeartbeatTime == nil {
		heatbeat := 3 * time.Minute
		s.HeartbeatTime = &heatbeat
	}

	return s.AlertLevel.checkValid()
}

func (e *EthereumBehindStrategy) getName() string {
	return "behind"
}

func (s *EthereumBehindStrategy) getAlertLevel() AlertLevel {
	return s.AlertLevel
}

func (e *EthereumBehindStrategy) initialize(logger *log.Entry, repo *repository.Repository) {
	e.logger = logger
	e.repo = repo
}

func (s *EthereumBehindStrategy) check(nodeName string) (alertEvent, string, bool) {
	_, _, fn := util.TraceFirst()
	s.logger.Tracef(fmt.Sprintf("Check function: %s", fn))

	latestEthBlockNumbers, err := s.repo.FindEthereumBlockNumberByAgentNameWithLimit(nodeName, time.Now().Add(-*s.HeartbeatTime), 100)

	if err != nil {
		s.logger.Error(errors.Wrap(err, "Error finding latest eth block number").Error())
		return EthBehindAlarmEvent, "", true
	}

	if len(latestEthBlockNumbers) == 0 {
		// there is no ethBlockNumbers
		return EthHeartbeatAlarmEvent, fmt.Sprintf("Heartbeat check failed (%v)", s.HeartbeatTime), true
	}

	var (
		currentBlockNumber    int64
		highestEthBlockNumber int64
	)

	currentBlockNumber, err = strconv.ParseInt(latestEthBlockNumbers[0].BlockNumber, 10, 64)
	if err != nil {
		s.logger.Error(errors.Wrap(err, "Error parsing block number").Error())
		return EthBehindAlarmEvent, "", true
	}
	highestEthBlockNumber = currentBlockNumber

	if s.APITimeout == time.Duration(0) {
		s.APITimeout = 10 * time.Second
	}
	rpc := JsonRPC{
		&http.Client{Timeout: s.APITimeout},
	}

	for _, apiUrl := range s.TargetRPCs {

		res := new(JsonRPCResponse)
		res, err = rpc.Call(apiUrl, "eth_blockNumber", []string{})
		if err != nil {
			s.logger.Error(errors.Wrapf(err, "Error calling eth_blockNumber: %s", apiUrl).Error())
			continue
		}

		var result string
		if err = json.Unmarshal(res.Result, &result); err != nil {
			s.logger.Error(errors.Wrapf(err, "Error unmarshalling result from eth_blockNumber: %s", apiUrl).Error())
			continue
		}

		var blockNumber int64
		if _, err = fmt.Sscanf(result, "0x%x", &blockNumber); err != nil {
			s.logger.Error(errors.Wrapf(err, "Error scanning result from eth_blockNumber: %s", apiUrl).Error())
			continue
		}

		if blockNumber > highestEthBlockNumber {
			highestEthBlockNumber = blockNumber
		}
	}

	if highestEthBlockNumber > currentBlockNumber {
		// currentBlockNumber is falling behind over s.MaxBehindBlockCount blocks.
		if highestEthBlockNumber-int64(s.MaxBehindBlockCount) > currentBlockNumber {
			return EthBehindAlarmEvent, fmt.Sprintf("EthBlockNumber is falling behind %d blocks.", highestEthBlockNumber-currentBlockNumber), true
		}
	} else { // monitoring node blockNumber is higher.

	}

	return EthBehindAlarmEvent, fmt.Sprintf("Complete to check Agents:(%s) currentBlockHeight: %d", nodeName, currentBlockNumber), false
}
