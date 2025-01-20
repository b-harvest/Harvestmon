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

	AlertEvent string `gorm:"column:alert_name;not null;type:varchar(100)"`
	Instance   string `gorm:"column:instance;not null;type:varchar(100)"`
	Target     string `gorm:"column:target;not null;type:varchar(100)"`
}

func (AlertRecord) TableName() string {
	return "alert_event_record"
}

func NewAlertRecord(startTs, resolvedTs *time.Time, target, alertEvent, instance string) (*AlertRecord, error) {
	// Generate a new UUID for the alert record
	alertRecordUUID, err := uuid.NewUUID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate UUID: %w", err)
	}

	return &AlertRecord{
		AlertRecordUUID: alertRecordUUID.String(),
		StartTimestamp:  startTs,
		Instance:        instance,
		Target:          target,
		ResolvTimestamp: resolvedTs,
		AlertEvent:      alertEvent,
	}, nil
}

func (r *Repository) UpdateResolvTs(alertRecord AlertRecord, resolveTs time.Time) error {
	if alertRecord.AlertRecordUUID == "" {
		return fmt.Errorf("alert_record_uuid is empty")
	}
	err := r.DB.Raw(`
UPDATE alert_event_record set resolv_timestamp = ? 
WHERE alert_record_uuid = ?`, resolveTs, alertRecord.AlertRecordUUID).Error
	if err != nil {
		return err
	}

	return nil
}

func (r *Repository) FindAlertRecordsByInstanceAndResolvTimestamp(instance string, resolveTimestamp *time.Time) ([]AlertRecord, error) {
	var result []AlertRecord

	if resolveTimestamp == nil {
		err := r.DB.Raw(`
SELECT *
FROM alert_event_record
WHERE instance = ?
AND resolv_timestamp is null
`, instance).Scan(&result).Error
		if err != nil {
			return nil, err
		}

	} else {
		err := r.DB.Raw(`
SELECT *
FROM alert_event_record
WHERE instance = ?
AND resolv_timestamp = ?
`, instance, *resolveTimestamp).Scan(&result).Error
		if err != nil {
			return nil, err
		}

	}

	return result, nil
}

func (r *Repository) FindAlertRecordsByResolvTimestamp(resolveTimestamp *time.Time) ([]AlertRecord, error) {
	var result []AlertRecord

	if resolveTimestamp == nil {
		err := r.DB.Raw(`
SELECT *
FROM alert_event_record
WHERE resolv_timestamp is null
`).Scan(&result).Error
		if err != nil {
			return nil, err
		}

	} else {
		err := r.DB.Raw(`
SELECT *
FROM alert_event_record
WHERE resolv_timestamp = ?
`, *resolveTimestamp).Scan(&result).Error
		if err != nil {
			return nil, err
		}

	}

	return result, nil
}
