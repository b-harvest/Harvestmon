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

func HeartbeatChecker(c *types.CheckerConfig, client *types.CheckerClient) {
	_, _, fn := util.TraceFirst()
	log.Debug(heartbeatFormatf("Starting: " + fn))

	eventRepository := repository.EventRepository{BaseRepository: repository.BaseRepository{DB: *client.GetDatabase(), CommitId: c.CommitId}}

	for agentName, agentChecker := range c.AgentCheckers {
		lastAgentNameAndCreatedAts, err := eventRepository.FindEventByServiceNameByAgentName(string(agentName), _const.HARVESTMON_TENDERMINT_SERVICE_NAME)
		if err != nil {
			log.Error(errors.New(heartbeatFormatf(err.Error())))
		}

		for _, event := range lastAgentNameAndCreatedAts {
			var now = time.Now().UTC()
			maxWaitTime, exists := (*agentChecker.Heartbeat)[event.EventType]
			if !exists {
				maxWaitTime = (*agentChecker.Heartbeat)[types.DefaultMaxWaitTimeKey]
			}
			if event.CreatedAt.Add(*maxWaitTime).Before(now) {

				var errorMsg = fmt.Sprintf("\nLatest Heartbeat: \n"+
					"%v (%v ago)\n\n"+
					"EventType: %s\n"+
					"ThresholdAlertHeartbeat: %v",
					event.CreatedAt, now.Sub(event.CreatedAt), event.EventType, *maxWaitTime)

				var (
					alertLevel types.AlertLevel
					sent       bool
				)

				if alertLevelP := client.GetAlertLevel(agentName, string(HEARTBEAT_TM_ALARM_TYPE), event.EventType); alertLevelP == nil {
					alertLevelP := client.GetAlertLevel(agentName, string(HEARTBEAT_TM_ALARM_TYPE))
					alertLevel = *alertLevelP
				} else {
					alertLevel = *alertLevelP
				}

				// Exceeded max missing count.

				for _, a := range client.GetAlarmerList(agentName, alertLevel.AlertLevel) {
					sent = true

					// Pass to alarmer
					err = alarmer.RunAlarm(c, *client, types.NewAlert(a, alertLevel, agentName, errorMsg))
					if err != nil {
						log.Error(errors.New(heartbeatFormatf("error occurred while sending alarm: %s, %v", HEARTBEAT_TM_ALARM_TYPE, err)))
					}
				}
				if !sent {
					log.Error(errors.New(heartbeatFormatf("Didn't send any alert cause of no alarmer specified for the level: %s, %s", HEARTBEAT_TM_ALARM_TYPE, alertLevel.AlertLevel)))
				}
			}
			log.Debug(heartbeatFormatf("Complete to check Agent: %s new event inserted at %v (%s ago)", event.AgentName, event.CreatedAt, time.Now().UTC().Sub(event.CreatedAt)))
		}
	}

}
