package repository

import (
	"gorm.io/gorm"
	"tendermint-mon/types"
	"time"
)

type Event struct {
	EventUUID   string    `gorm:"column:event_uuid;not null;type:UUID"`
	AgentName   string    `gorm:"column:agent_name;not null;type:varchar(100)"`
	ServiceName string    `gorm:"column:service_name;not null;type:varchar(100)"`
	CommitID    string    `gorm:"column:commit_id;not null;type:varchar(255)"`
	IsChecked   bool      `gorm:"column:is_checked;not null;type:bool,default:false"`
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

type EventRepository struct {
	CommitId string
	Db       gorm.DB
}

func (r *EventRepository) Save(event Event) error {
	res := r.Db.Create(&event)
	if res.Error != nil {
		return res.Error
	}

	return nil
}

type AgentEventWithCreatedAt struct {
	AgentName string    `gorm:"column:agent_name"`
	CreatedAt time.Time `gorm:"column:created_at;not null;type:datetime(6)"`
}

func (r *EventRepository) FindEventByServiceNameGroupByAgentName() ([]AgentEventWithCreatedAt, error) {
	var result []AgentEventWithCreatedAt

	err := r.Db.Raw(`select agent_name, max(created_at) as created_at
from event
where service_name = ?
group by agent_name;`, types.HARVEST_SERVICE_NAME).Scan(&result).Error
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (r *EventRepository) FindEventByServiceNameWithLimitGroupBydAgentName(serviceName string, limit int) []Event {

	return nil
}
