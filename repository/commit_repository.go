package repository

import (
	"github.com/b-harvest/Harvestmon/log"
	"gorm.io/gorm/schema"
	"time"
)

type TendermintCommit struct {
	CreatedAt          time.Time                   `gorm:"primaryKey;column:created_at;not null;type:datetime(6);autoCreateTime:false;autoCreateTime:false"`
	Event              Event                       `gorm:"foreignKey:EventUUID;references:EventUUID"`
	EventUUID          string                      `gorm:"primaryKey;column:event_uuid;not null;type:UUID"`
	ChainID            string                      `gorm:"column:chain_id;not null;type:varchar(20)"`
	Height             string                      `gorm:"column:height;not null;type:bigint"`
	Time               time.Time                   `gorm:"column:time;not null;type:datetime(6)"`
	LastBlockIdHash    string                      `gorm:"column:last_block_id_hash;not null;type:varchar(100)"`
	LastCommitHash     string                      `gorm:"column:last_commit_hash;not null;type:varchar(100)"`
	DataHash           string                      `gorm:"column:data_hash;not null;type:varchar(100)"`
	ValidatorsHash     string                      `gorm:"column:validators_hash;not null;type:varchar(100)"`
	NextValidatorsHash string                      `gorm:"column:next_validators_hash;not null;type:varchar(100)"`
	ConsensusHash      string                      `gorm:"column:consensus_hash;not null;type:varchar(100)"`
	AppHash            string                      `gorm:"column:app_hash;not null;type:varchar(100)"`
	LastResultsHash    string                      `gorm:"column:last_results_hash;not null;type:varchar(100)"`
	EvidenceHash       string                      `gorm:"column:evidence_hash;not null;type:varchar(100)"`
	ProposerAddress    string                      `gorm:"column:proposer_address;not null;type:varchar(100)"`
	Round              int32                       `gorm:"column:round;not null;type:int"`
	CommitBlockIdHash  string                      `gorm:"column:commit_block_id_hash;not null;type:varchar(100)"`
	Signatures         []TendermintCommitSignature `gorm:"foreignKey:TendermintCommitCreatedAt,EventUUID;references:CreatedAt,EventUUID"`
}

func (TendermintCommit) TableName() string {
	return "tendermint_commit"
}

type TendermintCommitSignature struct {
	ValidatorAddress          string           `gorm:"primaryKey;column:validator_address;not null;type:varchar(100)"`
	TendermintCommit          TendermintCommit `gorm:"foreignKey:TendermintCommitCreatedAt,EventUUID;references:CreatedAt,EventUUID"`
	TendermintCommitCreatedAt time.Time        `gorm:"primaryKey;column:tendermint_commit_created_at;not null;type:datetime(6);autoCreateTime:false"`
	Event                     Event            `gorm:"foreignKey:EventUUID;references:EventUUID"`
	EventUUID                 string           `gorm:"primaryKey;column:event_uuid;not null;type:UUID"`
	Timestamp                 time.Time        `gorm:"column:timestamp;not null;type:datetime(6)"`
	Signature                 string           `gorm:"column:signature;not null;type:varchar(200)"`
	BlockIdFlag               int              `gorm:"column:block_id_flag;not null;type:int"`
}

func (TendermintCommitSignature) TableName() string {
	return "tendermint_commit_signature"
}

type CommitRepository struct {
	EventRepository
}

func (r *CommitRepository) Save(tendermintCommit TendermintCommit) error {
	// Insert event
	//err := r.EventRepository.Save(event)
	//if err != nil {
	//	return err
	//}

	// Insert tendermint_commit_signature_list
	//res, err := r.Db.ExecContext(context.Background(), "INSERT INTO harvestmon.tendermint_commit (created_at, event_uuid, chain_id, height, time, last_block_id_hash, last_commit_hash, data_hash, validators_hash, next_validators_hash, consensus_hash, app_hash, last_results_hash, evidence_hash, proposer_address, round, commit_block_id_hash) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
	//	tendermintCommit.CreatedAt,
	//	tendermintCommit.EventUUID,
	//	tendermintCommit.ChainID,
	//	tendermintCommit.Height,
	//	tendermintCommit.Time,
	//	tendermintCommit.LastBlockIdHash,
	//	tendermintCommit.LastCommitHash,
	//	tendermintCommit.DataHash,
	//	tendermintCommit.ValidatorsHash,
	//	tendermintCommit.NextValidatorsHash,
	//	tendermintCommit.ConsensusHash,
	//	tendermintCommit.AppHash,
	//	tendermintCommit.LastResultsHash,
	//	tendermintCommit.EvidenceHash,
	//	tendermintCommit.ProposerAddress,
	//	tendermintCommit.Round,
	//	tendermintCommit.CommitBlockIdHash,
	//)
	//
	//if err != nil {
	//	return err
	//}
	//
	//_, err = res.RowsAffected()
	//if err != nil {
	//	return err
	//}
	//
	//// Insert tendermint_status
	//for _, signature := range tendermintCommit.Signatures {
	//	res, err = r.Db.ExecContext(context.Background(), "INSERT INTO harvestmon.tendermint_commit_signature_list (validator_address, created_at, event_uuid, timestamp, signature, block_id_flag) VALUES (?, ?, ?, ?, ?, ?)",
	//		signature.ValidatorAddress,
	//		tendermintCommit.CreatedAt,
	//		tendermintCommit.EventUUID,
	//		signature.Timestamp,
	//		signature.Signature,
	//		signature.BlockIdFlag,
	//	)
	//
	//	if err != nil {
	//		return err
	//	}
	//
	//	_, err = res.RowsAffected()
	//	if err != nil {
	//		return err
	//	}
	//}

	eventAssociation := r.Db.Model(&tendermintCommit).Association("Event")
	eventAssociation.Relationship.Type = schema.BelongsTo
	err := eventAssociation.Append(&tendermintCommit.Event)
	if err != nil {
		return err
	}

	res := r.Db.Create(&tendermintCommit)
	if res.Error != nil {
		return res.Error
	}

	log.Debug("Inserted `event`, `tendermint_commit`, `tendermint_commit_signature_list` successfully. eventUUID: " + tendermintCommit.Event.EventUUID)

	return nil
}

type TendermintCommitAndCommitSignaturesAndEvent struct {
	TendermintCommit
	Event
}

func (r *CommitRepository) FindCommitAndCommitSignaturesAndEventByServiceNameAndOrderByTimeDescAfterTimeWithLimitGroupByAgentName(serviceName string, timestamp time.Time, limit int) ([]TendermintCommitAndCommitSignaturesAndEvent, error) {
	//	rows, err := r.Db.QueryContext(context.Background(), `SELECT
	//    event.event_uuid,
	//    event.agent_name,
	//    event.service_name,
	//    event.commit_id,
	//    event.event_type,
	//    event.created_at,
	//    commit.created_at,
	//    commit.chain_id,
	//    commit.height,
	//    commit.time,
	//    commit.last_block_id_hash,
	//    commit.last_commit_hash,
	//    commit.data_hash,
	//    commit.validators_hash,
	//    commit.next_validators_hash,
	//    commit.consensus_hash,
	//    commit.app_hash,
	//    commit.last_results_hash,
	//    commit.evidence_hash,
	//    commit.proposer_address,
	//    commit.round,
	//    commit.commit_block_id_hash
	//FROM
	//    (SELECT
	//         event.agent_name,
	//         MAX(commit.time) AS time
	//     FROM
	//         event,
	//         tendermint_commit AS commit
	//     WHERE
	//         event.event_uuid = commit.event_uuid
	//       AND event.service_name = $1
	//       AND commit.time > str_to_date($2, '%Y-%m-%d %H:%i:%S.%f')
	//     GROUP BY
	//         event.agent_name
	//    ) AS maxCommit,
	//    event,
	//    tendermint_commit AS commit
	//WHERE
	//    commit.time = maxCommit.time
	//  AND commit.event_uuid = event.event_uuid
	//  AND event.event_type = $3;`, serviceName, "tm:event:commit", timestamp.Format("%Y-%m-%d %H:%i:%S.%f"))
	//
	//	if err != nil {
	//		return nil, err
	//	}
	//	defer rows.Close()
	//
	//	var commitAndCommitSignaturesAndEvents []TendermintCommitAndCommitSignaturesAndEvent
	//	for rows.Next() {
	//		var (
	//			// Event
	//			eventUUID         string
	//			agentName         string
	//			commitID          string
	//			eventType         string
	//			rawEventCreatedAt string
	//
	//			// Commit
	//			commitCreatedAt   string
	//			chainId           string
	//			height            string
	//			rawCommitTime     string
	//			lastBlockIdHash   string
	//			lastCommitHash    string
	//			dataHash          string
	//			validatorHash     string
	//			nextValidatorHash string
	//			consensusHash     string
	//			appHash           string
	//			lastResultsHash   string
	//			evidenceHash      string
	//			proposerAddress   string
	//			round             int32
	//			commitBlockIdHash string
	//		)
	//		err = rows.Scan(
	//			// Event
	//			&eventUUID,
	//			&agentName,
	//			&serviceName,
	//			&commitID,
	//			&eventType,
	//			&rawEventCreatedAt,
	//			// Commit
	//			&commitCreatedAt,
	//			&chainId,
	//			&height,
	//			&rawCommitTime,
	//			&lastBlockIdHash,
	//			&lastCommitHash,
	//			&dataHash,
	//			&validatorHash,
	//			&nextValidatorHash,
	//			&consensusHash,
	//			&appHash,
	//			&lastResultsHash,
	//			&evidenceHash,
	//			&proposerAddress,
	//			&round,
	//			&commitBlockIdHash)
	//
	//		eventCreatedAt, err := time.Parse(rawEventCreatedAt, "%Y-%m-%d %H:%i:%S.%f")
	//		if err != nil {
	//			return nil, err
	//		}
	//
	//		commitTime, err := time.Parse(rawCommitTime, "%Y-%m-%d %H:%i:%S.%f")
	//		if err != nil {
	//			return nil, err
	//		}
	//
	//		commitAndCommitSignaturesAndEvents = append(commitAndCommitSignaturesAndEvents, TendermintCommitAndCommitSignaturesAndEvent{
	//			Event: Event{
	//				EventUUID:   eventUUID,
	//				AgentName:   agentName,
	//				ServiceName: checker.TENDERMINT_SERVICE_NAME,
	//				CommitID:    commitID,
	//				EventType:   eventType,
	//				CreatedAt:   eventCreatedAt,
	//			},
	//			TendermintCommit: TendermintCommit{
	//				CreatedAt:          eventCreatedAt,
	//				EventUUID:          eventUUID,
	//				ChainID:            chainId,
	//				Height:             height,
	//				Time:               commitTime,
	//				LastBlockIdHash:    lastBlockIdHash,
	//				LastCommitHash:     lastCommitHash,
	//				DataHash:           dataHash,
	//				ValidatorsHash:     validatorHash,
	//				NextValidatorsHash: nextValidatorHash,
	//				ConsensusHash:      consensusHash,
	//				AppHash:            appHash,
	//				LastResultsHash:    lastResultsHash,
	//				EvidenceHash:       evidenceHash,
	//				ProposerAddress:    proposerAddress,
	//				Round:              round,
	//				CommitBlockIdHash:  commitBlockIdHash,
	//				Signatures:         []TendermintCommitSignature{},
	//			},
	//		})
	//
	//	}
	return nil, nil
}
