package monitor

import (
	log "github.com/b-harvest/Harvestmon/log"
	"github.com/google/uuid"
	"tendermint-mon/repository"
	"tendermint-mon/types"
	"tendermint-mon/util"
	"time"
)

func BlockCommitMonitor(c *types.MonitorConfig, client *types.MonitorClient) {
	_, _, fn := util.Trace()
	log.Debug("Starting monitor: " + fn)

	commitMonitorRepository := repository.CommitMonitorRepository{EventType: types.TM_COMMIT_EVENT_TYPE, Db: client.GetDatabase(), Agent: c.Agent}

	commit, err := client.GetCommit()
	if err != nil {
		log.Error(err)
	}

	eventUUID, err := uuid.NewUUID()
	if err != nil {
		log.Error(err)
	}

	createdAt := time.Now()

	var signatures []repository.TendermintCommitSignature
	for _, signature := range commit.Result.Commit.Signatures {
		signatures = append(signatures, repository.TendermintCommitSignature{
			ValidatorAddress: signature.ValidatorAddress,
			CreatedAt:        createdAt,
			EventUUID:        eventUUID.String(),
			Timestamp:        signature.Timestamp,
			Signature:        string(signature.Signature),
			BlockIdFlag:      signature.BlockIDFlag,
		})
	}

	err = commitMonitorRepository.Save(
		repository.Event{
			EventUUID:   eventUUID.String(),
			AgentName:   c.Agent.AgentName,
			ServiceName: types.HARVEST_SERVICE_NAME,
			CommitID:    c.Agent.CommitId,
			EventType:   types.TM_COMMIT_EVENT_TYPE,
			CreatedAt:   createdAt,
		},
		repository.TendermintCommit{
			CreatedAt:          createdAt,
			EventUUID:          eventUUID.String(),
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
			Signatures:         []repository.TendermintCommitSignature{},
		})
	if err != nil {
		log.Warn(err.Error())
	}

	log.Debug("Complete monitor: " + fn)
}
