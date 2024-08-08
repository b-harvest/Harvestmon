package checker

import (
	"errors"
	"github.com/b-harvest/Harvestmon/log"
	"github.com/b-harvest/Harvestmon/repository"
	"tendermint-checker/alarmer"
	"tendermint-checker/types"
	"time"
)

func HeartbeatChecker(c *types.CheckerConfig, client *types.CheckerClient) {
	eventRepository := repository.EventRepository{DB: *client.GetDatabase(), CommitId: c.CommitId}

	lastAgentNameAndCreatedAts, err := eventRepository.FindEventByServiceNameGroupByAgentName()
	if err != nil {
		log.Error(errors.New(heartbeatFormatf(err.Error())))
	}

	for _, event := range lastAgentNameAndCreatedAts {
		if event.CreatedAt.Add(c.Heartbeat.MaxWaitTime).Before(time.Now()) {
			if alertLevel, exists := client.AlertLevelList[HEARTBEAT_TM_ALARM_TYPE]; exists {
				for _, a := range alertLevel.AlarmerList {

					// Pass to alarmer
					alarmer.RunAlarm(alarmer.NewAlert(a))
				}

			} else {
				log.Error(errors.New(heartbeatFormatf("Cannot find alarm level: %s", HEARTBEAT_TM_ALARM_TYPE)))
			}
		}
		log.Debug(heartbeatFormatf("Complete to check Agent: %s new event inserted at %v (%s ago)", event.AgentName, event.CreatedAt, time.Now().Sub(event.CreatedAt)))
	}

}
