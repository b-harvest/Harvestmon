package repository

import (
	"fmt"
	"gorm.io/gorm"
	"time"
)

type EthereumBlockNumber struct {
	CreatedAt   time.Time `gorm:"primaryKey;column:created_at;not null;type:datetime(6);autoCreateTime:false"`
	Event       Event     `gorm:"foreignKey:EventUUID;references:EventUUID"`
	EventUUID   string    `gorm:"primaryKey;column:event_uuid;not null;type:CHAR(36)"`
	BlockNumber string    `gorm:"column:block_number;not null;type:bigint"`
}

func (s *EthereumBlockNumber) BeforeCreate(tx *gorm.DB) (err error) {
	err = tx.Create(&s.Event).Error
	return
}

func (EthereumBlockNumber) TableName() string {
	return "ethereum_block_number"
}

func (r *Repository) FindLatestEthBlockNumbersByAgentName(agentName, eventType, serviceName string, createdAt time.Time, count int) ([]EthereumBlockNumber, error) {
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
