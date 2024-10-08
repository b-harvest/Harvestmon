package checker

import (
	"errors"
	"fmt"
	"github.com/b-harvest/Harvestmon/checker/tendermint/alarmer"
	"github.com/b-harvest/Harvestmon/checker/tendermint/types"
	"github.com/b-harvest/Harvestmon/log"
	"github.com/b-harvest/Harvestmon/repository"
	"github.com/b-harvest/Harvestmon/util"
)

func BlockCommitChecker(c *types.CheckerConfig, client *types.CheckerClient) {
	_, _, fn := util.TraceFirst()
	log.Debug(blockCommitFormatf("Starting monitor: " + fn))

	commitRepository := repository.CommitRepository{BaseRepository: repository.BaseRepository{DB: *client.GetDatabase(), CommitId: c.CommitId}}

	for agentName, agentChecker := range c.AgentCheckers {
		if agentChecker.CommitCheck.ValidatorAddress == "" {
			log.Debug(blockCommitFormatf("Skipping block commitment check... agent: %s", agentName))
			continue
		}

		// T.C.(TendermintCommit)
		validatorAddressesWithAgents, err := commitRepository.FindValidatorAddressesWithAgents(
			agentChecker.CommitCheck.ValidatorAddress,
			agentChecker.CommitCheck.TargetBlockCount,
			string(agentName))
		if err != nil {
			log.Error(errors.New(blockCommitFormatf(err.Error())))
			return
		}

		var (
			agentWithSignCounts = make(map[types.AgentName]int)
			isBreakRows         = make(map[types.AgentName]bool)
		)

		for _, validatorAddressesWithAgent := range validatorAddressesWithAgents {

			// Actually, it doesn't matter to check it is matching with validator address
			// because `validatorAddressesWithAgent.ValidatorAddress` is same with `c.CommitCheck.ValidatorAddress`
			//
			// When fetching validatorAddessesWithAgents, it'll automatically filter validator addresses if it's not target address.
			// but also get least one row even its field is nil.
			if validatorAddressesWithAgent.ValidatorAddress == agentChecker.CommitCheck.ValidatorAddress {
				agentWithSignCounts[agentName]++
			}
		}

		if agentChecker.CommitCheck.TargetBlockCount-agentWithSignCounts[agentName] > agentChecker.CommitCheck.MaxMissingCount {
			if isBreakRows[agentName] {
				log.Debug(blockCommitFormatf("Agent(%s)'s commit records have been break. to check signing infos, it should be over than %d. ignoring...", agentName, agentChecker.CommitCheck.TargetBlockCount))
				continue
			}

			var errorMsg = fmt.Sprintf("\nWatching block until: %d blocks, SignCount: %d\nThresholdMissingCount: %d",
				agentChecker.CommitCheck.TargetBlockCount, agentWithSignCounts[agentName], agentChecker.CommitCheck.MaxMissingCount)

			var (
				alertLevel types.AlertLevel
				sent       bool
			)

			if alertLevelP := client.GetAlertLevel(agentName, string(MISSING_BLOCK_TM_ALARM_TYPE)); alertLevelP == nil {
				log.Error(errors.New(blockCommitFormatf("alertLevel not found: %s", string(MISSING_BLOCK_TM_ALARM_TYPE))))
			} else {
				alertLevel = *alertLevelP
			}

			// Exceeded max missing count.

			for _, a := range client.GetAlarmerList(agentName, alertLevel.AlertLevel) {
				sent = true
				// Pass to alarmer
				err = alarmer.RunAlarm(c, *client, types.NewAlert(a, alertLevel, agentName, errorMsg))
				if err != nil {
					log.Error(errors.New(blockCommitFormatf("error occurred while sending alarm: %s, %v", MISSING_BLOCK_TM_ALARM_TYPE, err)))
				}
			}
			if !sent {
				log.Error(errors.New(blockCommitFormatf("Didn't send any alert cause of no alarmer specified for the level: %s, %s", MISSING_BLOCK_TM_ALARM_TYPE, alertLevel.AlertLevel)))
			}

		}
		log.Debug(blockCommitFormatf("Complete to check Agents:(%s) signing block count: %d", agentName, agentWithSignCounts[agentName]))
	}

}
