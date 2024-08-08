package checker

import (
	"errors"
	"github.com/b-harvest/Harvestmon/log"
	"github.com/b-harvest/Harvestmon/repository"
	"tendermint-checker/alarmer"
	"tendermint-checker/types"
	"time"
)

type Uint64AndBool struct {
	uint64
	bool
}

func HeightStuckChecker(c *types.CheckerConfig, client *types.CheckerClient) {

	// Check if it is stuck
	statusRepository := repository.StatusRepository{EventRepository: repository.EventRepository{DB: *client.GetDatabase(), CommitId: c.CommitId}}
	startTime := time.Now().Add(-c.HeightCheck.MaxStuckTime)
	tsEvents, err := statusRepository.FindTSEventsAfterStartTimeGroupByAgentName(startTime)
	if err != nil {
		log.Error(errors.New(heightCheckFormatf(err.Error())))
	}

	var checkAgentMap = make(map[string]Uint64AndBool)

	// Judge height stuck
	// latestTendermintStatuses is mixed with other agents. so it'll calculate using map[string;agentName]Uint64AndBool.
	for _, status := range tsEvents {
		if agent, exists := checkAgentMap[status.AgentName]; exists {

			// Check if changed.
			if agent.uint64 != status.LatestBlockHeight {
				agent.bool = true
				continue
			}
		} else {

			// Set new height when there are no values mapped.
			checkAgentMap[status.AgentName] = Uint64AndBool{uint64: status.LatestBlockHeight, bool: false}
		}
	}

	for key, check := range checkAgentMap {
		if !check.bool {
			if alertLevel, exists := client.AlertLevelList[HEIGHT_STUCK_TM_ALARM_TYPE]; exists {
				for _, a := range alertLevel.AlarmerList {

					// Pass to alarmer
					alarmer.RunAlarm(alarmer.NewAlert(a))
				}

			} else {
				log.Error(errors.New(heightCheckFormatf("Cannot find alarm level: %s", HEIGHT_STUCK_TM_ALARM_TYPE)))
			}
		}
		log.Debug(heightCheckFormatf("Complete to check Agent: (%s). height changed = %t.(latestHeight: %d)", key, check.bool, check.uint64))
	}

}
