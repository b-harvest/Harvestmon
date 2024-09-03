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
	EventUUID          string                      `gorm:"primaryKey;column:event_uuid;not null;type:CHAR(36)"`
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
	EventUUID                 string           `gorm:"primaryKey;column:event_uuid;not null;type:CHAR(36)"`
	Timestamp                 time.Time        `gorm:"column:timestamp;not null;type:datetime(6)"`
	Signature                 string           `gorm:"column:signature;not null;type:varchar(200)"`
	BlockIdFlag               int              `gorm:"column:block_id_flag;not null;type:int"`
}

func (TendermintCommitSignature) TableName() string {
	return "tendermint_commit_signature"
}

type CommitRepository struct {
	BaseRepository
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

func (r *CommitRepository) CreateBatch(tendermintCommits []TendermintCommit) error {
	var events []Event
	for _, tendermintCommit := range tendermintCommits {
		events = append(events, tendermintCommit.Event)
	}

	eventRepository := EventRepository{BaseRepository: r.BaseRepository}
	err := eventRepository.CreateBatch(events)
	if err != nil {
		return err
	}

	res := r.DB.CreateInBatches(tendermintCommits, len(tendermintCommits))
	if res.Error != nil {
		return res.Error
	}

	log.Debug("Inserted batch slices for `event`, `tendermint_commit`, `tendermint_commit_signature_list` successfully.")

	return nil
}

func (r *CommitRepository) FetchHighestHeight(agentName, commitId string) (uint64, error) {
	var (
		maxHeight uint64
	)
	err := r.DB.Model(&TendermintCommit{}).
		Joins("JOIN event ON event.event_uuid = tendermint_commit.event_uuid").
		Where("event.agent_name = ? AND event.commit_id = ?", agentName, commitId).
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

func (r *CommitRepository) FindValidatorAddressesWithAgents(validatorAddress string, limit int, agentName string) ([]ValidatorAddressesWithAgents, error) {

	var result []ValidatorAddressesWithAgents
	err := r.DB.Raw(`SELECT
    e.agent_name,
    tc.event_uuid,
    tc.created_at,
    tc.height,
    tcs.validator_address
FROM
    event e
        LEFT JOIN tendermint_commit tc ON e.event_uuid = tc.event_uuid
        LEFT JOIN tendermint_commit_signature tcs
                  ON tc.event_uuid = tcs.event_uuid
                      AND tc.created_at = tcs.tendermint_commit_created_at
                      AND tcs.validator_address = ?
WHERE e.commit_id = ?
  AND e.agent_name = ?
AND e.created_at 
ORDER BY
    e.agent_name DESC,
    tc.height DESC
LIMIT ?;
`, validatorAddress, r.CommitId, agentName, limit).Scan(&result).Error

	if err != nil {
		return nil, err
	}

	return result, nil

}

func (r *CommitRepository) FindValidatorAddressesWithAgentsUsingStartTime(validatorAddress string, startTime time.Time) ([]ValidatorAddressesWithAgents, error) {

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
    tendermint_commit_signature tcs 
    ON tc.event_uuid = tcs.event_uuid
    AND tc.created_at = tcs.tendermint_commit_created_at
    AND tcs.validator_address = ?
WHERE 
    tc.created_at >= ?
    AND e.commit_id = ?
    AND EXISTS (
        SELECT 1
        FROM tendermint_commit tc_inner
        JOIN event e_inner ON tc_inner.event_uuid = e_inner.event_uuid
        WHERE 
            tc_inner.created_at = tc.created_at
            AND e_inner.agent_name = e.agent_name
    )
ORDER BY 
    e.agent_name DESC, 
    tc.height DESC
`, validatorAddress, startTime, r.CommitId).Scan(&result).Error

	if err != nil {
		return nil, err
	}

	return result, nil
}
