package monitor

import (
	"fmt"
	"github.com/b-harvest/Harvestmon/log"
	"github.com/b-harvest/Harvestmon/repository"
	"github.com/b-harvest/Harvestmon/util"
	"github.com/google/uuid"
	"strconv"
	"tendermint-mon/types"
	"time"
)

func BlockCommitMonitor(c *types.MonitorConfig, client *types.MonitorClient) {
	_, _, fn := util.Trace()
	log.Debug("Starting monitor: " + fn)

	commitMonitorRepository := repository.CommitRepository{EventRepository: repository.EventRepository{DB: *client.GetDatabase()}}

	status, err := client.GetCometBFTStatus()
	if err != nil {
		log.Error(err)
	}
	latestHeight, err := strconv.ParseUint(status.SyncInfo.LatestBlockHeight, 0, 64)
	if err != nil {
		log.Error(err)
	}

	startHeight, err := commitMonitorRepository.FetchHighestHeight(c.Agent.AgentName)
	if err != nil {
		log.Warn(err.Error())
		startHeight = latestHeight - 1
	} else {
		// Start after latest stored commit height.
		startHeight++
	}

	if (latestHeight - startHeight) > (uint64(c.Agent.PushInterval.Seconds()) * 5) {
		startHeight = latestHeight - (uint64(c.Agent.PushInterval.Seconds()) * 5)
		log.Info(fmt.Sprintf("[block_commit] distance from startHeight to latestHeight is too large. automatically set startHeight as %d", startHeight))
	}

	for i := startHeight; i < latestHeight; i++ {
		commit, err := client.GetCommitWithHeight(i)
		if err != nil {
			log.Error(err)
		}

		eventUUID, err := uuid.NewUUID()
		if err != nil {
			log.Error(err)
		}

		createdAt := time.Now()

		var signatures []repository.TendermintCommitSignature

		for _, signature := range commit.Result.SignedHeader.Commit.Signatures {
			if signature.ValidatorAddress == "" {
				continue
			}
			signatures = append(signatures, repository.TendermintCommitSignature{
				ValidatorAddress:          signature.ValidatorAddress,
				TendermintCommitCreatedAt: createdAt,
				EventUUID:                 eventUUID.String(),
				Timestamp:                 signature.Timestamp,
				Signature:                 signature.Signature,
				BlockIdFlag:               signature.BlockIDFlag,
			})
		}

		err = commitMonitorRepository.Save(
			repository.TendermintCommit{
				CreatedAt: createdAt,
				EventUUID: eventUUID.String(),
				Event: repository.Event{
					EventUUID:   eventUUID.String(),
					AgentName:   c.Agent.AgentName,
					ServiceName: types.HARVEST_SERVICE_NAME,
					CommitID:    c.Agent.CommitId,
					EventType:   types.TM_COMMIT_EVENT_TYPE,
					CreatedAt:   createdAt,
				},
				ChainID:            commit.Result.ChainID,
				Height:             commit.Result.Height,
				Time:               commit.Result.Time,
				LastBlockIdHash:    commit.Result.LastBlockID.Hash,
				LastCommitHash:     commit.Result.LastCommitHash,
				DataHash:           commit.Result.DataHash,
				ValidatorsHash:     commit.Result.ValidatorsHash,
				NextValidatorsHash: commit.Result.NextValidatorsHash,
				ConsensusHash:      commit.Result.ConsensusHash,
				AppHash:            commit.Result.AppHash,
				LastResultsHash:    commit.Result.LastResultsHash,
				EvidenceHash:       commit.Result.EvidenceHash,
				ProposerAddress:    commit.Result.ProposerAddress,
				Round:              commit.Result.Commit.Round,
				CommitBlockIdHash:  commit.Result.Commit.BlockID.Hash,
				Signatures:         signatures,
			})

		log.Info(fmt.Sprintf("[block_commit] height: %v, signature count: %d", i, len(signatures)))
		if err != nil {
			log.Warn(err.Error())
		}

	}

	log.Debug("Complete monitor: " + fn)
}
