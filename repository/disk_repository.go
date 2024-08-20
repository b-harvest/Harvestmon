package repository

import (
	"errors"
	"fmt"
	"github.com/b-harvest/Harvestmon/log"
	"gorm.io/gorm/schema"
	"time"
)

type DiskUsage struct {
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

func (DiskUsage) TableName() string {
	return "disk_usage"
}

type DiskUsageRepository struct {
	BaseRepository
}

func (r *DiskUsageRepository) Save(diskUsage DiskUsage) error {
	eventAssociation := r.DB.Model(&diskUsage).Association("Event")
	eventAssociation.Relationship.Type = schema.BelongsTo
	err := eventAssociation.Append(&diskUsage.Event)
	if err != nil {
		return err
	}

	res := r.DB.Create(&diskUsage)
	if res.Error != nil {
		return res.Error
	}

	log.Debug("Inserted `event`, `tendermint_commit`, `tendermint_commit_signature_list` successfully. eventUUID: " + diskUsage.Event.EventUUID)

	return nil
}

func (r *DiskUsageRepository) FetchHighestHeight(agentName string) (uint64, error) {
	var (
		maxHeight uint64
	)
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

func (r *DiskUsageRepository) FindValidatorAddressesWithAgents(validatorAddress string, limit int, agentName string) ([]ValidatorAddressesWithAgents, error) {

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
WHERE e.commit_id = ?
  and e.agent_name = ?
    and (e.agent_name, tc.created_at) IN (
        SELECT
            e_inner.agent_name,
            tc_inner.created_at
        FROM
            tendermint_commit tc_inner
                JOIN
            event e_inner ON tc_inner.event_uuid = e_inner.event_uuid
        WHERE e_inner.agent_name = ?
            and 
            (SELECT COUNT(*)
             FROM tendermint_commit tc_inner2
                      JOIN event e_inner2 ON tc_inner2.event_uuid = e_inner2.event_uuid
             WHERE e_inner2.agent_name = e_inner.agent_name
               AND tc_inner2.created_at >= tc_inner.created_at) <= ?
) ORDER BY agent_name desc, tc.height desc;
`, validatorAddress, r.CommitId, agentName, agentName, limit).Scan(&result).Error

	if err != nil {
		return nil, err
	}

	return result, nil

}

func (r *DiskUsageRepository) FindValidatorAddressesWithAgentsUsingStartTime(validatorAddress string, startTime time.Time) ([]ValidatorAddressesWithAgents, error) {

	var result []ValidatorAddressesWithAgents
	err := r.DB.Raw(`SELECT e.agent_name, tc.event_uuid, tc.created_at, tc.height, tcs.validator_address
FROM tendermint_commit tc
         JOIN event e ON tc.event_uuid = e.event_uuid
         LEFT JOIN tendermint_commit_signature tcs ON tc.event_uuid = tcs.event_uuid
    AND tc.created_at = tcs.tendermint_commit_created_at
    AND tcs.validator_address = ?
WHERE (e.agent_name, tc.created_at) IN (
    SELECT e_inner.agent_name, tc_inner.created_at
    FROM tendermint_commit tc_inner
             JOIN event e_inner ON tc_inner.event_uuid = e_inner.event_uuid
    WHERE tc_inner.created_at >= ?
    GROUP BY e_inner.agent_name, tc_inner.created_at
) and e.commit_id = ?
ORDER BY e.agent_name DESC, tc.height DESC;
`, validatorAddress, startTime, r.CommitId).Scan(&result).Error

	if err != nil {
		return nil, err
	}

	return result, nil
}
