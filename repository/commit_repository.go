package repository

import (
	"errors"
	"fmt"
	"github.com/b-harvest/Harvestmon/log"
	"gorm.io/gorm/schema"
	"time"
)

type TendermintCommit struct {
	CreatedAt          time.Time                   `gorm:"primaryKey;column:created_at;not null;type:datetime(6)"`
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
	TendermintCommitCreatedAt time.Time        `gorm:"primaryKey;column:tendermint_commit_created_at;not null;type:datetime(6)"`
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
	eventAssociation := r.DB.Model(&tendermintCommit).Association("Event")
	eventAssociation.Relationship.Type = schema.BelongsTo
	err := eventAssociation.Append(&tendermintCommit.Event)
	if err != nil {
		return err
	}

	res := r.DB.Create(&tendermintCommit)
	if res.Error != nil {
		return res.Error
	}

	log.Debug("Inserted `event`, `tendermint_commit`, `tendermint_commit_signature_list` successfully. eventUUID: " + tendermintCommit.Event.EventUUID)

	return nil
}

func (r *CommitRepository) FetchHighestHeight(agentName string) (uint64, error) {
	var maxHeight uint64
	err := r.DB.Model(&TendermintCommit{}).
		Joins("JOIN event ON event.event_uuid = tendermint_commit.event_uuid").
		Where("event.agent_name = ?", agentName).
		Select("MAX(tendermint_commit.height)").
		Scan(&maxHeight).Error

	if err != nil {
		return 0, errors.New(fmt.Sprintf("failed to get maximum height: %v", err))
	}

	return maxHeight, nil
}

type ValidatorAddressesWithAgents struct {
	AgentName        string    `gorm:"column:agent_name"`
	EventUUID        string    `gorm:"column:event_uuid"`
	CreatedAt        time.Time `gorm:"column:created_at;not null;type:datetime(6)"`
	Height           uint64    `gorm:"column:height"`
	ValidatorAddress string    `gorm:"column:validator_address;null"`
}

func (r *CommitRepository) FindValidatorAddressesWithAgents(validatorAddress string, limit int) ([]ValidatorAddressesWithAgents, error) {

	var result []ValidatorAddressesWithAgents
	err := r.DB.Raw(`SELECT
    e.agent_name,
    tc.event_uuid,
    tc.created_at,
    tc.height,
    tcs.validator_address
FROM
    tendermint_commit tc
        JOIN
    event e ON tc.event_uuid = e.event_uuid
        LEFT JOIN
    tendermint_commit_signature tcs ON tc.event_uuid = tcs.event_uuid
        AND tc.created_at = tcs.tendermint_commit_created_at
        AND tcs.validator_address = ?
WHERE
    (e.agent_name, tc.created_at) IN (
        SELECT
            e_inner.agent_name,
            tc_inner.created_at
        FROM
            tendermint_commit tc_inner
                JOIN
            event e_inner ON tc_inner.event_uuid = e_inner.event_uuid
        WHERE
            (SELECT COUNT(*)
             FROM tendermint_commit tc_inner2
                      JOIN event e_inner2 ON tc_inner2.event_uuid = e_inner2.event_uuid
             WHERE e_inner2.agent_name = e_inner.agent_name
               AND tc_inner2.created_at >= tc_inner.created_at) <= ?
) ORDER BY agent_name desc, tc.height desc;
`, validatorAddress, limit).Scan(&result).Error

	if err != nil {
		return nil, err
	}

	return result, nil

	//AND commit.time > str_to_date($2, '%Y-%m-%d %H:%i:%S.%f')
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
}
