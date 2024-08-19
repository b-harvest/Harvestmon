package repository

import (
	"gorm.io/gorm"
	"time"
)

type Event struct {
	EventUUID   string    `gorm:"primaryKey;column:event_uuid;not null;type:CHAR(36)"`
	AgentName   string    `gorm:"column:agent_name;not null;type:varchar(100)"`
	ServiceName string    `gorm:"column:service_name;not null;type:varchar(100)"`
	CommitID    string    `gorm:"column:commit_id;not null;type:varchar(255)"`
	EventType   string    `gorm:"column:event_type;not null;type:varchar(100)"`
	CreatedAt   time.Time `gorm:"column:created_at;not null;type:datetime(6);autoCreateTime:false"`

	TendermintCommits          []TendermintCommit          `gorm:"foreignKey:EventUUID;references:EventUUID"`
	TendermintCommitSignatures []TendermintCommitSignature `gorm:"foreignKey:EventUUID;references:EventUUID"`

	TendermintNetInfos  []TendermintNetInfo  `gorm:"foreignKey:EventUUID;references:EventUUID"`
	TendermintPeerInfos []TendermintPeerInfo `gorm:"foreignKey:EventUUID;references:EventUUID"`

	TendermintStatuses []TendermintStatus `gorm:"foreignKey:EventUUID;references:EventUUID"`
}

func (Event) TableName() string {
	return "event"
}

type MonitorRepository interface {
	Save(any ...any) error
}

type BaseRepository struct {
	CommitId string
	DB       gorm.DB
}

type EventRepository struct {
	BaseRepository
}

func (r *EventRepository) Save(event Event) error {
	res := r.DB.Create(&event)
	if res.Error != nil {
		return res.Error
	}

	return nil
}

type AgentEventWithCreatedAt struct {
	AgentName string    `gorm:"column:agent_name"`
	CreatedAt time.Time `gorm:"column:created_at;not null;type:datetime(6)"`
}

func (r *EventRepository) FindEventByServiceNameByAgentName(agentName, serviceName string) ([]AgentEventWithCreatedAt, error) {
	var result []AgentEventWithCreatedAt

	err := r.DB.Raw(`select agent_name, max(created_at) as created_at
from event
where service_name = ?
and commit_id = ?
and agent_name = ?
group by agent_name;`, serviceName, r.CommitId, agentName).Scan(&result).Error
	if err != nil {
		return nil, err
	}

	return result, nil
}
