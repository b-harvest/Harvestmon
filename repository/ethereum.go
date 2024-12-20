package repository

import (
	"fmt"
	"gorm.io/gorm/schema"
	"time"
)

type EthereumBlockNumber struct {
	CreatedAt   time.Time `gorm:"primaryKey;column:created_at;not null;type:datetime(6);autoCreateTime:false"`
	Event       Event     `gorm:"foreignKey:EventUUID;references:EventUUID"`
	EventUUID   string    `gorm:"primaryKey;column:event_uuid;not null;type:CHAR(36)"`
	BlockNumber string    `gorm:"column:block_number;not null;type:bigint"`
}

func (EthereumBlockNumber) TableName() string {
	return "ethereum_block_number"
}

func (s *EthereumBlockNumber) getEvent() *Event {
	return &s.Event
}

type EthBlockNumberRepository struct {
	Repository
}

func (r *EthBlockNumberRepository) Save(blockNumber EthereumBlockNumber) error {
	eventAssociation := r.DB.Model(&blockNumber).Association("Event")
	eventAssociation.Relationship.Type = schema.BelongsTo
	err := eventAssociation.Append(&blockNumber.Event)
	if err != nil {
		return err
	}

	res := r.DB.Create(&blockNumber)
	if res.Error != nil {
		return res.Error
	}
	return nil
}

func (r *EthBlockNumberRepository) FindLatestEthBlockNumbersByAgentName(agentName, eventType, serviceName string, createdAt time.Time, count int) ([]EthereumBlockNumber, error) {
	var result []EthereumBlockNumber

	err := r.DB.Raw(`SELECT /*+ JOIN_ORDER(e, sync) */
        sync.created_at, sync.block_number
    FROM
        event e, ethereum_block_number AS sync
    WHERE
        e.event_type = ? AND
        e.agent_name = ? AND
        e.service_name = ? AND
        e.commit_id = ? AND
        e.event_uuid = sync.event_uuid AND
        e.created_at >= ?
    ORDER BY
        e.created_at DESC
    LIMIT ?`, eventType, agentName, serviceName, r.CommitId, createdAt, count).Scan(&result).Error

	if err != nil {
		return nil, fmt.Errorf("failed to fetch Ethereum block numbers: %w", err)
	}

	return result, nil
}

func (r *EthBlockNumberRepository) SaveAll(blockNumbers []EthereumBlockNumber) error {
	if len(blockNumbers) == 0 {
		return nil
	}

	var events []Event
	for _, blockNumber := range blockNumbers {
		events = append(events, blockNumber.Event)
	}

	if err := r.DB.Create(&events).Error; err != nil {
		return err
	}

	if err := r.DB.Create(&blockNumbers).Error; err != nil {
		return err
	}

	return nil
}
