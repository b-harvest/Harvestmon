package main

import (
	"fmt"
	"github.com/b-harvest/Harvestmon/repository"
	"github.com/b-harvest/Harvestmon/util"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"time"
)

type TendermintHeightStrategy struct {
	logger *log.Entry

	Enabled       bool           `toml:"enabled"`
	MaxStuckTime  *time.Duration `toml:"maxStuckTime"`
	AlertLevel    AlertLevel     `toml:"alertLevel"`
	HeartbeatTime *time.Duration `toml:"heartbeatTime"` // if not set, defaults to 10 minutes.

	repo *repository.Repository
}

func (t *TendermintHeightStrategy) checkValid() error {
	if !t.Enabled {
		return nil
	}
	if t.MaxStuckTime == nil {
		return errors.New("maxStuckTime is empty")
	}
	if t.HeartbeatTime == nil {
		heatbeat := 10 * time.Minute
		t.HeartbeatTime = &heatbeat
	}

	return t.AlertLevel.checkValid()
}

func (c *TendermintHeightStrategy) getName() string {
	return "height"
}

func (s *TendermintHeightStrategy) getAlertLevel() AlertLevel {
	return s.AlertLevel
}

func (s *TendermintHeightStrategy) initialize(logger *log.Entry, repo *repository.Repository) {
	s.logger = logger
	s.repo = repo
}

func (s *TendermintHeightStrategy) check(nodeName string) (alertEvent, string, bool) {
	_, _, fn := util.TraceFirst()
	s.logger.Tracef(fmt.Sprintf("Check function: %s", fn))

	tsEvent, err := s.repo.FindTmStatusVOtFirstByAgentNameAndCreatedAtGreaterThanEqual(nodeName, time.Now().Add(-*s.HeartbeatTime))
	if err != nil {
		s.logger.Error(errors.Wrap(err, "error finding TS events by agent name").Error())
		return TendermintStuckAlarmEvent, "", true
	}

	if tsEvent == nil { // == no event detected -> hearbeat error
		return TendermintHeartbeatAlarmEvent, fmt.Sprintf("Heartbeat check failed (%v)", s.HeartbeatTime), true
	}

	if tsEvent.LatestBlockTime.Add(*s.MaxStuckTime).Before(time.Now().UTC()) {

		return TendermintStuckAlarmEvent, fmt.Sprintf("LatestBlock: %d, \ntime: %v(about %s)\nThresholdStuckTime: %v",
			tsEvent.LatestBlockHeight, tsEvent.LatestBlockTime.Format(time.DateTime), time.Now().Sub(tsEvent.LatestBlockTime).Round(time.Minute).String(), s.MaxStuckTime), true
	}
	return TendermintStuckAlarmEvent, fmt.Sprintf("Complete to check Agent: (%s).\nheight increasing.(latestHeight: %d)", nodeName, tsEvent.LatestBlockHeight), false
}

type TendermintPeerStrategy struct {
	logger *log.Entry

	Enabled       bool           `toml:"enabled"`
	LowPeerCount  uint           `toml:"lowPeerCount"`
	AlertLevel    AlertLevel     `toml:"alertLevel"`
	HeartbeatTime *time.Duration `toml:"heartbeatTime"` // if not set, defaults to 3 minutes
	repo          *repository.Repository
}

func (t *TendermintPeerStrategy) checkValid() error {
	if !t.Enabled {
		return nil
	}
	if t.LowPeerCount < 1 {
		return errors.New("low peer count must be greater than zero")
	}

	if t.HeartbeatTime == nil {
		heatbeat := 10 * time.Minute
		t.HeartbeatTime = &heatbeat
	}
	return t.AlertLevel.checkValid()
}

func (c *TendermintPeerStrategy) getName() string {
	return "peer"
}

func (s *TendermintPeerStrategy) getAlertLevel() AlertLevel {
	return s.AlertLevel
}

func (s *TendermintPeerStrategy) initialize(logger *log.Entry, repo *repository.Repository) {
	s.logger = logger
	s.repo = repo
}

func (s *TendermintPeerStrategy) check(nodeName string) (alertEvent, string, bool) {
	_, _, fn := util.TraceFirst()
	s.logger.Tracef(fmt.Sprintf("Check function: %s", fn))

	agentPeerInfos, err := s.repo.FindTmPeerVOsLatestByAgentNameAndCreatedAtGreaterThanEqual(nodeName, time.Now().Add(-*s.HeartbeatTime))
	if err != nil {
		s.logger.Error(fmt.Sprintf("error finding latest peer info by agent name: %s", nodeName))
		return TendermintPeerAlarmEvent, "", true
	}

	if len(agentPeerInfos) == 0 {
		return TendermintHeartbeatAlarmEvent, fmt.Sprintf("Heartbeat check failed (%v)", s.HeartbeatTime), true
	}

	for _, agentPeerInfo := range agentPeerInfos {
		if agentPeerInfo.CreatedAt.Add(5 * time.Minute).Before(time.Now().UTC()) {
			s.logger.Warning(fmt.Sprintf("Agent(%s)'s latest peer info is too old: %v (%s ago)", agentPeerInfo.AgentName, agentPeerInfo.CreatedAt, time.Now().Sub(agentPeerInfo.CreatedAt)))
		}
		if agentPeerInfo.NPeers != agentPeerInfo.PeerInfoUUIDCount {
			s.logger.Warningf(fmt.Sprintf("It is different with NPeers and length of PeerInfos. You should check it. EventUUID: %s", agentPeerInfo.EventUUID))
		}
		if agentPeerInfo.NPeers < int(s.LowPeerCount) {
			return TendermintPeerAlarmEvent, fmt.Sprintf("Current Peer Count: %d\nThresholdPeer: %d", agentPeerInfo.NPeers, s.LowPeerCount), true
		}

		return TendermintPeerAlarmEvent, fmt.Sprintf("Complete to check Agent:(%s)\npeers: %d", agentPeerInfo.AgentName, agentPeerInfo.NPeers), false
	}

	return TendermintPeerAlarmEvent, "", true
}

type TendermintCommitStrategy struct {
	logger *log.Entry

	Enabled          bool           `toml:"enabled"`
	ValidatorAddress string         `toml:"validatorAddress"`
	MaxMissingCount  uint           `toml:"maxMissingCount"`
	TargetBlockCount uint           `toml:"targetBlockCount"`
	AlertLevel       AlertLevel     `toml:"alertLevel"`
	HeartbeatTime    *time.Duration `toml:"heartbeatTime"` // if not set, defaults to 10 minutes.

	repo *repository.Repository
}

func (t *TendermintCommitStrategy) checkValid() error {
	if !t.Enabled {
		return nil
	}
	if t.ValidatorAddress != "" {
		if t.TargetBlockCount < 1 {
			return errors.New("target block count must be greater than zero")
		}
	}
	if t.HeartbeatTime == nil {
		heatbeat := 10 * time.Minute
		t.HeartbeatTime = &heatbeat
	}

	return t.AlertLevel.checkValid()
}

func (c *TendermintCommitStrategy) getName() string {
	return "commit"
}

func (s *TendermintCommitStrategy) getAlertLevel() AlertLevel {
	return s.AlertLevel
}

func (s *TendermintCommitStrategy) initialize(logger *log.Entry, repo *repository.Repository) {
	s.logger = logger
	s.repo = repo
}

func (s *TendermintCommitStrategy) check(nodeName string) (alertEvent, string, bool) {
	_, _, fn := util.TraceFirst()
	s.logger.Tracef("Check function: " + fn)

	validatorAddressesWithAgents, err := s.repo.FindTmCommitVOsWithAgents(
		s.ValidatorAddress,
		int(s.TargetBlockCount),
		nodeName)
	if err != nil {
		s.logger.Error(errors.Wrap(err, "Error finding validator addresses with agents").Error())
		return TendermintCommitAlarmEvent, "", true
	}

	var (
		signingCnt int
	)

	if len(validatorAddressesWithAgents) == 0 ||
		validatorAddressesWithAgents[0].CreatedAt.Add(*s.HeartbeatTime).Before(time.Now()) {
		return TendermintHeartbeatAlarmEvent, fmt.Sprintf("Heartbeat check failed (%v)", s.HeartbeatTime), true
	}

	if len(validatorAddressesWithAgents) >= int(s.TargetBlockCount) {
		for _, validatorAddressesWithAgent := range validatorAddressesWithAgents {

			// Actually, it doesn't matter to check it is matching with validator address
			// because `validatorAddressesWithAgent.ValidatorAddress` is same with `c.CommitCheck.ValidatorAddress`
			//
			// When fetching validatorAddessesWithAgents, it'll automatically filter validator addresses if it's not target address.
			// but also get least one row even its field is nil.
			if validatorAddressesWithAgent.ValidatorAddress == s.ValidatorAddress {
				signingCnt++
			}
		}
		if int(s.TargetBlockCount)-signingCnt > int(s.MaxMissingCount) {

			return TendermintCommitAlarmEvent, fmt.Sprintf("Missed validator: %s\nwatching period: %d blocks, \nsigns: %d\n threshold: %d",
				s.ValidatorAddress, s.TargetBlockCount, signingCnt, s.MaxMissingCount), true

		}
	} else {
		s.logger.Warningf("Agent(%s)'s commit records are not enough. to check signing infos, it should be over than %d. ignoring...", nodeName, s.TargetBlockCount)
		return TendermintCommitAlarmEvent, "", false
	}

	return TendermintCommitAlarmEvent, fmt.Sprintf("Complete to check Agents:(%s)\nsigning block count: %d", nodeName, signingCnt), false
}
