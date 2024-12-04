package checker

import (
	"errors"
	"fmt"
	"github.com/b-harvest/Harvestmon/checker/ethereum/alarmer"
	"github.com/b-harvest/Harvestmon/checker/ethereum/types"
	_const "github.com/b-harvest/Harvestmon/const"
	"github.com/b-harvest/Harvestmon/log"
	"github.com/b-harvest/Harvestmon/repository"
	"github.com/b-harvest/Harvestmon/util"
	"time"
)

func BlockNumberChecker(c *types.CheckerConfig, client *types.CheckerClient) {
	_, _, fn := util.TraceFirst()
	log.Debug(blockNumberFormatf("Starting monitor: " + fn))

	ethBlockNumberRepository := repository.EthBlockNumberRepository{BaseRepository: repository.BaseRepository{DB: *client.GetRDatabase(), CommitId: c.CommitId}}

	for agentName, agentChecker := range c.AgentCheckers {
		if agentChecker == nil || agentChecker.EthBlockCheck == nil {
			log.Debug(blockNumberFormatf("Skipping ethBlockNumber check... agent: %s", agentName))
			continue
		}

		latestEthBlockNumbers, err := ethBlockNumberRepository.FindLatestEthBlockNumbersByAgentName(string(agentName), _const.ETH_BLOCK_NUMBER_EVENT_TYPE, _const.HARVESTMON_ETHEREUM_SERVICE_NAME, 100)

		if err != nil {
			log.Error(errors.New(blockNumberFormatf(err.Error())))
			return
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
				if ethBlockNumber.BlockNumber == initBlockNumber.BlockNumber {
					beforeChangedBlockNumberTime = ethBlockNumber.CreatedAt
				} else {
					break
				}
			}
		}

		// no ethBlockNumbers detected
		if len(latestEthBlockNumbers) == 0 {
			log.Debug(blockNumberFormatf("Skipping ethBlockNumber check... agent: %s", agentName))
		}

		// height doesn't change for a while
		if (time.Time{}.Equal(beforeChangedBlockNumberTime)) || currentBlockNumberTime.Sub(beforeChangedBlockNumberTime) > *agentChecker.EthBlockCheck.MaxStuckTime {

			var errorMsg = fmt.Sprintf("\nEthBlockNumber has stuck for a while.\nBlockNumber: %s \nLastBlockTime: %s \nMaxStuckTime: %s",
				initBlockNumber.BlockNumber, beforeChangedBlockNumberTime.String(), agentChecker.EthBlockCheck.MaxStuckTime)

			var (
				alertLevel types.AlertLevel
				sent       bool
			)

			if alertLevelP := client.GetAlertLevel(agentName, string(STUCK_ETH_ALARM_TYPE)); alertLevelP == nil {
				log.Error(errors.New(blockNumberFormatf("alertLevel not found: %s", string(STUCK_ETH_ALARM_TYPE))))
			} else {
				alertLevel = *alertLevelP
			}

			for _, a := range client.GetAlarmerList(agentName, alertLevel.AlertLevel) {
				sent = true
				// Pass to alarmer
				err = alarmer.RunAlarm(c, *client, types.NewAlert(a, alertLevel, agentName, errorMsg))
				if err != nil {
					log.Error(errors.New(blockNumberFormatf("error occurred while sending alarm: %s, %v", STUCK_ETH_ALARM_TYPE, err)))
				}
			}
			if !sent {
				log.Error(errors.New(blockNumberFormatf("Didn't send any alert cause of no alarmer specified for the level: %s, %s", STUCK_ETH_ALARM_TYPE, alertLevel.AlertLevel)))
			}
		}

		log.Debug(blockNumberFormatf("Complete to check Agents:(%s) new block height: %s", agentName, initBlockNumber.BlockNumber))
	}

}
