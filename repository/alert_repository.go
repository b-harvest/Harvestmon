package repository

import (
	"github.com/b-harvest/Harvestmon/log"
	"time"
)

type AlertRecord struct {
	AlertRecordUUID string `gorm:"primaryKey;column:alert_record_uuid;not null;type:CHAR(36)"`

	CreatedAt   time.Time `gorm:"column:alert_record_created_at;not null;type:datetime(6)"`
	AlertName   string    `gorm:"column:alert_name;not null;type:varchar(100)"`
	LevelName   string    `gorm:"column:level_name;not null;type:varchar(100)"`
	AlarmerName string    `gorm:"column:alarmer_name;not null;type:varchar(255)"`

	AgentName string `gorm:"column:agent_name;not null;type:varchar(100)"`
	CommitID  string `gorm:"column:commit_id;not null;type:varchar(255)"`
}

func (AlertRecord) TableName() string {
	return "alert_record"
}

type AlertRecordRepository struct {
	BaseRepository
}

func (r *AlertRecordRepository) Save(alertRecord AlertRecord) error {
	res := r.DB.Create(&alertRecord)
	if res.Error != nil {
		return res.Error
	}

	log.Debug("Inserted `alert_record` successfully. alertRecordUUID: " + alertRecord.AlertRecordUUID)

	return nil
}

func (r *AlertRecordRepository) ExistsIfAlertRecordIsMarkedOrAlreadySent(alertName, alarmerName, agentName string, startTime, endTime time.Time, maxMarkDuration time.Duration) (bool, error) {
	var (
		result           bool
		now              = time.Now().UTC()
		maxMarkStartTime = now.Add(-maxMarkDuration)
	)

	err := r.DB.Raw(`select (
           exists(select 1
     from alert_record as ar
     WHERE ar.alert_name = ?
       AND ar.alarmer_name = ?
       AND ar.agent_name = ?
       AND ar.commit_id = ?
       and ar.alert_record_created_at >= ?
       AND ar.alert_record_created_at < ?)
     or
           exists(select 1
     from agent_mark as m
     where (m.agent_name = ?
         and (
                (m.mark_end is not null and m.mark_end >= ?)
                    or
                (m.mark_end is null and m.mark_start >= ?)
                )
         and m.mark_start <= ?))
)

`, alertName, alarmerName, agentName, r.CommitId, startTime, endTime, agentName, endTime, maxMarkStartTime, endTime).Scan(&result).Error

	if err != nil {
		return false, err
	}

	return result, nil
}
