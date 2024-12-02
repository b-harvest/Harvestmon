package repository

import (
	"gorm.io/gorm/schema"
	"time"
)

type EthereumIsSyncing struct {
	CreatedAt time.Time `gorm:"primaryKey;column:created_at;not null;type:datetime(6);autoCreateTime:false"`
	Event     Event     `gorm:"foreignKey:EventUUID;references:EventUUID"`
	EventUUID string    `gorm:"primaryKey;column:event_uuid;not null;type:CHAR(36)"`
	IsSyncing bool      `gorm:"column:is_syncing;not null;type:boolean"`
}

func (EthereumIsSyncing) TableName() string {
	return "ethereum_is_syncing"
}

type EthIsSyncingRepository struct {
	BaseRepository
}

func (r *EthIsSyncingRepository) Save(isSyncing EthereumIsSyncing) error {
	eventAssociation := r.DB.Model(&isSyncing).Association("Event")
	eventAssociation.Relationship.Type = schema.BelongsTo
	err := eventAssociation.Append(&isSyncing.Event)
	if err != nil {
		return err
	}

	res := r.DB.Create(&isSyncing)
	if res.Error != nil {
		return res.Error
	}
	return nil
}

func (r *EthIsSyncingRepository) FindLatestIsSyncingsByAgentName(agentName, eventType, serviceName string, count int) ([]EthereumIsSyncing, error) {
	var result []EthereumIsSyncing

	err := r.DB.Raw(`SELECT
    sync.*
FROM
    event e
JOIN ethereum_is_syncing as sync
on e.event_uuid = sync.event_uuid
where e.event_type = ?
and e.agent_name = ?
and e.service_name = ?
and e.commit_id = ?
order by e.created_at desc
limit 100`, eventType, agentName, serviceName, r.CommitId, count).Scan(&result).Error

	if err != nil {
		return nil, err
	}

	return result, nil
}
