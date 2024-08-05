package repository

import (
	"context"
	"github.com/adlio/schema"
	log "github.com/b-harvest/Harvestmon/harvestmon-log"
	"tendermint-mon/types"
	"time"
)

type TendermintCommit struct {
	CreatedAt          time.Time
	EventUUID          string
	ChainID            string
	Height             string
	Time               time.Time
	LastBlockIdHash    string
	LastCommitHash     string
	DataHash           string
	ValidatorsHash     string
	NextValidatorsHash string
	ConsensusHash      string
	AppHash            string
	LastResultsHash    string
	EvidenceHash       string
	ProposerAddress    string
	Round              int32
	CommitBlockIdHash  string
	Signatures         []TendermintCommitSignature
}

type TendermintCommitSignature struct {
	ValidatorAddress string
	CreatedAt        time.Time
	EventUUID        string
	Timestamp        time.Time
	Signature        string
	BlockIdFlag      int
}

type CommitMonitorRepository struct {
	Db        schema.Queryer
	EventType string
	Agent     types.MonitoringAgent
}

func (r *CommitMonitorRepository) Save(event Event, tendermintCommit TendermintCommit) error {
	// Insert event
	res, err := r.Db.ExecContext(context.Background(), "INSERT INTO harvestmon.event (event_uuid, agent_name, service_name, commit_id, event_type, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		event.EventUUID, event.AgentName, event.ServiceName, event.CommitID, event.EventType, event.CreatedAt)
	if err != nil {
		return err
	}

	_, err = res.RowsAffected()
	if err != nil {
		return err
	}

	// Insert tendermint_commit_signature_list
	res, err = r.Db.ExecContext(context.Background(), "INSERT INTO harvestmon.tendermint_commit (created_at, event_uuid, chain_id, height, time, last_block_id_hash, last_commit_hash, data_hash, validators_hash, next_validators_hash, consensus_hash, app_hash, last_results_hash, evidence_hash, proposer_address, round, commit_block_id_hash) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		tendermintCommit.CreatedAt,
		tendermintCommit.EventUUID,
		tendermintCommit.ChainID,
		tendermintCommit.Height,
		tendermintCommit.Time,
		tendermintCommit.LastBlockIdHash,
		tendermintCommit.LastCommitHash,
		tendermintCommit.DataHash,
		tendermintCommit.ValidatorsHash,
		tendermintCommit.NextValidatorsHash,
		tendermintCommit.ConsensusHash,
		tendermintCommit.AppHash,
		tendermintCommit.LastResultsHash,
		tendermintCommit.EvidenceHash,
		tendermintCommit.ProposerAddress,
		tendermintCommit.Round,
		tendermintCommit.CommitBlockIdHash,
	)

	if err != nil {
		return err
	}

	_, err = res.RowsAffected()
	if err != nil {
		return err
	}

	// Insert tendermint_status
	for _, signature := range tendermintCommit.Signatures {
		res, err = r.Db.ExecContext(context.Background(), "INSERT INTO harvestmon.tendermint_commit_signature_list (validator_address, created_at, event_uuid, timestamp, signature, block_id_flag) VALUES (?, ?, ?, ?, ?, ?)",
			signature.ValidatorAddress,
			tendermintCommit.CreatedAt,
			tendermintCommit.EventUUID,
			signature.Timestamp,
			signature.Signature,
			signature.BlockIdFlag,
		)

		if err != nil {
			return err
		}

		_, err = res.RowsAffected()
		if err != nil {
			return err
		}
	}

	log.Debug("Inserted `event`, `tendermint_commit`, `tendermint_commit_signature_list` successfully. eventUUID: " + event.EventUUID)

	return nil
}
