package monitor

import (
	"errors"
	"fmt"
	_const "github.com/b-harvest/Harvestmon/const"
	"github.com/b-harvest/Harvestmon/log"
	"github.com/b-harvest/Harvestmon/monitor/elmon/types"
	"github.com/b-harvest/Harvestmon/repository"
	"github.com/b-harvest/Harvestmon/util"
	"github.com/google/uuid"
	"time"
)

func BlockNumberMonitor(c *types.MonitorConfig, client *types.MonitorClient) {
	_, _, fn := util.TraceFirst()
	log.Debug("Starting monitor: " + fn)

	blockNumberRepository := repository.EthBlockNumberRepository{BaseRepository: repository.BaseRepository{DB: *client.GetDatabase(c.DbBatchSize)}}

	eventUUID, err := uuid.NewUUID()
	if err != nil {
		log.Error(err)
	}
	createdAt := time.Now().UTC()

	blockNumber, err := client.GetBlockNumber()
	if err != nil {
		log.Error(errors.New(fmt.Sprintf("Error getting block number: %s", err)))
		return
	}

	err = blockNumberRepository.Save(
		repository.EthereumBlockNumber{
			CreatedAt: createdAt,
			EventUUID: eventUUID.String(),
			Event: repository.Event{
				EventUUID:   eventUUID.String(),
				AgentName:   c.Agent.AgentName,
				ServiceName: _const.HARVESTMON_ETHEREUM_SERVICE_NAME,
				CommitID:    c.Agent.CommitId,
				EventType:   _const.ETH_BLOCK_NUMBER_EVENT_TYPE,
				CreatedAt:   createdAt,
			},
			BlockNumber: fmt.Sprintf("%d", blockNumber),
		})
	if err != nil {
		log.Warn(err.Error())
	}

	log.Info(fmt.Sprintf("Block number: %d", blockNumber))

	log.Debug("Completed monitor: " + fn)
}
