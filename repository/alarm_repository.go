package repository

import (
	"fmt"
)

// it only stores activeAlarm.
// just for cache.
type ActiveAlarm struct {
	SentTime int64 `gorm:"column:alert_record_sent_time;not null;type:bigint"`

	AlarmerName string `gorm:"primaryKey;column:alarmer_name;not null;type:varchar(100)"`

	Target   string `gorm:"primaryKey;column:target;not null;type:varchar(100)"`
	Instance string `gorm:"primaryKey;column:instance;not null;type:varchar(100)"`
}

func (ActiveAlarm) TableName() string {
	return "active_alarm"
}

func NewActiveAlarm(sentAt int64, alarmerName, target, instance string) (*ActiveAlarm, error) {
	return &ActiveAlarm{
		SentTime:    sentAt,
		AlarmerName: alarmerName,
		Target:      target,
		Instance:    instance,
	}, nil
}

func (r *Repository) FindActiveAlarmsByInstance(instance string) ([]ActiveAlarm, error) {
	var result []ActiveAlarm
	err := r.DB.Raw(`SELECT *
FROM 
    active_alarm
WHERE instance = ?
`, instance).Scan(&result).Error

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (r *Repository) FindActiveAlarms() ([]ActiveAlarm, error) {
	var result []ActiveAlarm
	err := r.DB.Raw(`SELECT *
FROM 
    active_alarm
`).Scan(&result).Error

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (r *Repository) UpdateActiveAlarmSentTime(alarm ActiveAlarm, ts int64) error {

	err := r.DB.Exec(`
UPDATE active_alarm
SET alert_record_sent_time = ?
WHERE alarmer_name = ?
AND instance = ?
AND target = ?
`, ts, alarm.AlarmerName, alarm.Instance, alarm.Target).Error
	if err != nil {
		return err
	}

	return nil
}

func (r *Repository) DeleteActiveAlarms(alist []ActiveAlarm) error {
	if len(alist) == 0 {
		return nil
	}

	// Generate the conditions for a raw SQL query
	var params []interface{}
	query := "DELETE FROM active_alarm WHERE (instance, target, alarmer_name) IN ("

	for i, alarm := range alist {
		if i > 0 {
			query += ","
		}
		query += "(?, ?, ?)"
		params = append(params, alarm.Instance, alarm.Target, alarm.AlarmerName)
	}

	query += ")"

	// Execute the raw SQL query
	if err := r.DB.Exec(query, params...).Error; err != nil {
		return fmt.Errorf("failed to delete alarms: %w", err)
	}

	return nil
}
