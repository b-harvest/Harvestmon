package repository

import (
	"fmt"
	"github.com/google/uuid"
	"time"
)

type AlertRecord struct {
	AlertRecordUUID string `gorm:"primaryKey;column:alert_record_uuid;not null;type:CHAR(36)"`

	StartTimestamp  *time.Time `gorm:"column:start_timestamp;not null;type:DATETIME"`
	ResolvTimestamp *time.Time `gorm:"column:resolv_timestamp;null;type:DATETIME"`

	AlertEvent     string `gorm:"column:alert_name;not null;type:varchar(100)"`
	NodeName       string `gorm:"column:node_name;not null;type:varchar(100)"`
	StrategyTarget string `gorm:"column:strategy_target;not null;type:varchar(100)"`

	CommitID string `gorm:"column:commit_id;not null;type:varchar(255)"`
}

func (AlertRecord) TableName() string {
	return "alert_event_record"
}

func NewAlertRecord(startTs, resolvedTs *time.Time, strategyTarget, alertEvent, node, commitId string) (*AlertRecord, error) {
	// Generate a new UUID for the alert record
	alertRecordUUID, err := uuid.NewUUID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate UUID: %w", err)
	}

	return &AlertRecord{
		AlertRecordUUID: alertRecordUUID.String(),
		StartTimestamp:  startTs,
		NodeName:        node,
		StrategyTarget:  strategyTarget,
		ResolvTimestamp: resolvedTs,
		AlertEvent:      alertEvent,
		CommitID:        commitId,
	}, nil
}

func (r *Repository) UpdateResolvTs(alertRecord AlertRecord, resolveTs time.Time) error {
	err := r.DB.Exec(`
UPDATE alert_event_record set resolv_timestamp = ? 
WHERE alert_record_uuid = ?`, resolveTs, alertRecord.AlertRecordUUID).Error
	if err != nil {
		return err
	}

	return nil
}

func (r *Repository) FindAlertRecordsByNodeNameAndResolvTimestamp(nodeName string, resolveTimestamp *time.Time) ([]AlertRecord, error) {
	var result []AlertRecord

	if resolveTimestamp == nil {
		err := r.DB.Raw(`
SELECT *
FROM alert_event_record
WHERE node_name = ?
AND resolv_timestamp is null
`, nodeName).Scan(&result).Error
		if err != nil {
			return nil, err
		}

	} else {
		err := r.DB.Raw(`
SELECT *
FROM alert_event_record
WHERE node_name = ?
AND resolv_timestamp = ?
`, nodeName, *resolveTimestamp).Scan(&result).Error
		if err != nil {
			return nil, err
		}

	}

	return result, nil
}
