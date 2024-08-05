package monitor

import (
	"errors"
	log "github.com/b-harvest/harvestmon-log"
	"github.com/google/uuid"
	"strconv"
	"tendermint-mon/repository"
	"tendermint-mon/types"
	"tendermint-mon/util"
	"time"
)

func CometBFTStatusMonitor(c *types.MonitorConfig, client *types.MonitorClient) {
	_, _, fn := util.Trace()
	log.Debug("Starting monitor: " + fn)

	statusMonitorRepository := repository.StatusMonitorRepository{EventType: types.TM_STATUS_EVENT_TYPE, Db: client.GetDatabase(), Agent: c.Agent}

	cometBFTStatus, err := client.GetCometBFTStatus()
	if err != nil {
		log.Error(err)
	}

	eventUUID, err := uuid.NewUUID()
	if err != nil {
		log.Error(err)
	}
	nodeInfoUUID, err := uuid.NewUUID()
	if err != nil {
		log.Error(err)
	}

	createdAt := time.Now()

	latestBlockHeight, err := strconv.ParseUint(cometBFTStatus.SyncInfo.LatestBlockHeight, 0, 64)
	earliestBlockHeight, err := strconv.ParseUint(cometBFTStatus.SyncInfo.EarliestBlockHeight, 0, 64)
	if err != nil {
		log.Error(errors.New("Parsing error: " + cometBFTStatus.SyncInfo.LatestBlockHeight + ", " + cometBFTStatus.SyncInfo.EarliestBlockHeight + ". err: " + err.Error()))
	}

	err = statusMonitorRepository.Save(
		repository.Event{
			EventUUID:   eventUUID.String(),
			AgentName:   c.Agent.AgentName,
			ServiceName: types.HARVEST_SERVICE_NAME,
			CommitID:    c.Agent.CommitId,
			EventType:   types.TM_STATUS_EVENT_TYPE,
			CreatedAt:   createdAt,
		},
		repository.TendermintNodeInfo{
			TendermintNodeInfoUUID: nodeInfoUUID.String(),
			NodeId:                 string(cometBFTStatus.NodeInfo.DefaultNodeID),
			ListenAddr:             cometBFTStatus.NodeInfo.ListenAddr,
			ChainId:                cometBFTStatus.NodeInfo.Network,
			Moniker:                cometBFTStatus.NodeInfo.Moniker,
		},
		repository.TendermintStatus{
			CreatedAt:              createdAt,
			EventUUID:              eventUUID.String(),
			TendermintNodeInfoUUID: nodeInfoUUID.String(),
			LatestBlockHash:        string(cometBFTStatus.SyncInfo.LatestBlockHash),
			LatestAppHash:          string(cometBFTStatus.SyncInfo.LatestAppHash),
			LatestBlockHeight:      latestBlockHeight,
			LatestBlockTime:        cometBFTStatus.SyncInfo.LatestBlockTime,
			EarliestBlockHash:      string(cometBFTStatus.SyncInfo.EarliestBlockHash),
			EarliestAppHash:        string(cometBFTStatus.SyncInfo.EarliestAppHash),
			EarliestBlockHeight:    earliestBlockHeight,
			EarliestBlockTime:      cometBFTStatus.SyncInfo.EarliestBlockTime,
			CatchingUp:             cometBFTStatus.SyncInfo.CatchingUp,
		})
	if err != nil {
		log.Warn(err.Error())
	}

	log.Debug("Complete monitor: " + fn)
}
