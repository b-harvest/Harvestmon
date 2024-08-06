package repository

import (
	"gorm.io/gorm"
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
	// Insert event
	//res, err := r.Db.ExecContext(context.Background(), "INSERT INTO harvestmon.event (event_uuid, agent_name, service_name, commit_id, event_type, created_at) VALUES (?, ?, ?, ?, ?, ?)",
	//	event.EventUUID, event.AgentName, event.ServiceName, event.CommitID, event.EventType, event.CreatedAt)
	//if err != nil {
	//	return err
	//}
	//
	//_, err = res.RowsAffected()
	//if err != nil {
	//	return err
	//}
	res := r.Db.Create(&event)
	if res.Error != nil {
		return res.Error
	}

	return nil
}

func (r *EventRepository) FindEventByServiceNameGroupByAgentName(serviceName string) []Event {
	return nil
}

func (r *EventRepository) FindEventByServiceNameWithLimitGroupBydAgentName(serviceName string, limit int) []Event {

	return nil
}
