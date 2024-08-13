package checker

import (
	"errors"
	"github.com/b-harvest/Harvestmon/log"
	"github.com/b-harvest/Harvestmon/repository"
	"tendermint-checker/alarmer"
	"tendermint-checker/types"
)

func BlockCommitChecker(c *types.CheckerConfig, client *types.CheckerClient) {
	commitRepository := repository.CommitRepository{EventRepository: repository.EventRepository{DB: *client.GetDatabase(), CommitId: c.CommitId}}

	// T.C.(TendermintCommit)
	validatorAddressesWithAgents, err := commitRepository.FindValidatorAddressesWithAgents(c.CommitCheck.ValidatorAddress, c.CommitCheck.TargetBlockCount)
	if err != nil {
		log.Error(errors.New(blockCommitFormatf(err.Error())))
		return
	}

	var (
		agentWithSignCounts = make(map[string]int)
		agentBeforeHeight   = make(map[string]uint64)
		agentNames          []string
		IsBreakRows         = make(map[string]bool)
	)

	for _, validatorAddressesWithAgent := range validatorAddressesWithAgents {
		if agentBeforeHeight[validatorAddressesWithAgent.AgentName] == 0 {
			agentNames = append(agentNames, validatorAddressesWithAgent.AgentName)
			agentBeforeHeight[validatorAddressesWithAgent.AgentName] = validatorAddressesWithAgent.Height
		}

		// Check if this row is connected with before height.
		if agentBeforeHeight[validatorAddressesWithAgent.AgentName] != validatorAddressesWithAgent.Height-1 || IsBreakRows[validatorAddressesWithAgent.AgentName] {
			IsBreakRows[validatorAddressesWithAgent.AgentName] = true

			// Actually, it doesn't matter to check it is matching with validator address
			// because `validatorAddressesWithAgent.ValidatorAddress` is same with `c.CommitCheck.ValidatorAddress`
			//
			// When fetching validatorAddessesWithAgents, it'll automatically filter validator addresses if it's not target address.
			// but also get least one row even its field is nil.
		} else if validatorAddressesWithAgent.ValidatorAddress == c.CommitCheck.ValidatorAddress {
			agentWithSignCounts[validatorAddressesWithAgent.AgentName]++
			agentBeforeHeight[validatorAddressesWithAgent.AgentName] = validatorAddressesWithAgent.Height
		}

	}

	for _, agentName := range agentNames {
		if c.CommitCheck.TargetBlockCount-agentWithSignCounts[agentName] > c.CommitCheck.MaxMissingCount {
			if IsBreakRows[agentName] {
				log.Debug(blockCommitFormatf("Agent(%s)'s commit records have been break. to check signing infos, it should be over than %d. ignoring...", agentName, c.CommitCheck.TargetBlockCount))
				continue
			}
			// Exceeded max missing count.
			if alertLevel, exists := client.AlertLevelList[MISSING_BLOCK_TM_ALARM_TYPE]; exists {
				for _, a := range alertLevel.AlarmerList {

					// Pass to alarmer
					alarmer.RunAlarm(alarmer.NewAlert(a))
				}

			} else {
				log.Error(errors.New(blockCommitFormatf("Cannot find alarm level: %s", MISSING_BLOCK_TM_ALARM_TYPE)))
			}
		}
		log.Debug(blockCommitFormatf("Complete to check Agents:(%s) missing block count: %d", agentName, agentWithSignCounts[agentName]))
	}

}
