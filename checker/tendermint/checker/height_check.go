package checker

import (
	"errors"
	"fmt"
	"github.com/b-harvest/Harvestmon/checker/tendermint/alarmer"
	"github.com/b-harvest/Harvestmon/checker/tendermint/types"
	_const "github.com/b-harvest/Harvestmon/const"
	"github.com/b-harvest/Harvestmon/log"
	"github.com/b-harvest/Harvestmon/repository"
	"github.com/b-harvest/Harvestmon/util"
	"time"
)

type Uint64BoolFirstTime struct {
	uint64
	bool
	time.Time
}

func HeightStuckChecker(c *types.CheckerConfig, client *types.CheckerClient) {
	_, _, fn := util.TraceFirst()
	log.Debug(heightCheckFormatf("Starting: " + fn))

	// Check if it is stuck
	statusRepository := repository.StatusRepository{BaseRepository: repository.BaseRepository{DB: *client.GetDatabase(), CommitId: c.CommitId}}

	for agentName, agentChecker := range c.AgentCheckers {
		startTime := time.Now().UTC().Add(-*agentChecker.HeightCheck.MaxStuckTime)
		tsEvents, err := statusRepository.FindTSEventsAfterStartTimeGroupByAgentName(startTime, string(agentName), _const.HARVESTMON_TENDERMINT_SERVICE_NAME)
		if err != nil {
			log.Error(errors.New(heightCheckFormatf(err.Error())))
		}

		if tsEvents == nil {
			log.Error(errors.New(heightCheckFormatf("No TendermintStatuses found after %v", startTime)))
			continue
		}

		var checkAgent = Uint64BoolFirstTime{}

		// Judge height stuck
		// latestTendermintStatuses is mixed with other agents. so it'll utilize map[string;agentName]Uint64AndBool.
		for _, status := range tsEvents {
			if checkAgent.uint64 == 0 {
				checkAgent = Uint64BoolFirstTime{uint64: status.LatestBlockHeight, bool: false, Time: status.LatestBlockTime}
			} else {
				// Check if changed.
				if checkAgent.uint64 != status.LatestBlockHeight {
					checkAgent = Uint64BoolFirstTime{
						status.LatestBlockHeight,
						true,
						status.LatestBlockTime,
					}
					continue
				}
			}
		}

		if !checkAgent.bool {

			var errorMsg = fmt.Sprintf("\nLatestBlock: \n height: %d, time: %v(stuck in %v)\nThresholdStuckTime: %v",
				checkAgent.uint64, checkAgent.Time, time.Now().Sub(checkAgent.Time), agentChecker.HeightCheck.MaxStuckTime)

			var (
				alertLevel = client.GetAlertLevelList(agentName, string(HEIGHT_STUCK_TM_ALARM_TYPE))
				sent       bool
			)
			// Exceeded max missing count.

			for _, a := range client.GetAlarmerList(agentName, alertLevel.AlertLevel) {
				sent = true

				// Pass to alarmer
				err = alarmer.RunAlarm(c, *client, types.NewAlert(a, alertLevel, agentName, errorMsg))
				if err != nil {
					log.Error(errors.New(heightCheckFormatf("error occurred while sending alarm: %s, %v", HEIGHT_STUCK_TM_ALARM_TYPE, err)))
				}
			}
			if !sent {
				log.Error(errors.New(heightCheckFormatf("Didn't send any alert cause of no alarmer specified for the level: %s, %s", HEIGHT_STUCK_TM_ALARM_TYPE, alertLevel.AlertLevel)))
			}

		}
		log.Debug(heightCheckFormatf("Complete to check Agent: (%s). height changed = %t.(latestHeight: %d)", agentName, checkAgent.bool, checkAgent.uint64))
	}

}
