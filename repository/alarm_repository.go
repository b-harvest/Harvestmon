package repository

import (
	"fmt"
)

// it only stores activeAlarm.
// just for cache.
type ActiveAlarm struct {
	SentTime int64 `gorm:"column:alert_record_sent_time;not null;type:bigint"`

	AlarmerName string `gorm:"column:alarmer_name;not null;type:varchar(100)"`

	StrategyTarget string `gorm:"column:strategy_target;not null;type:varchar(100)"`
	NodeName       string `gorm:"column:node_name;not null;type:varchar(100)"`
	CommitID       string `gorm:"column:commit_id;not null;type:varchar(255)"`
}

func (ActiveAlarm) TableName() string {
	return "active_alarm"
}

func NewActiveAlarm(sentAt int64, alarmerName, strategyTarget, nodeName, commitId string) (*ActiveAlarm, error) {
	return &ActiveAlarm{
		SentTime:       sentAt,
		AlarmerName:    alarmerName,
		StrategyTarget: strategyTarget,
		NodeName:       nodeName,
		CommitID:       commitId,
	}, nil
}

type ActiveAlarmRepository struct {
	BaseRepository
}

func (r *ActiveAlarmRepository) Save(alarmRecord ActiveAlarm) error {
	res := r.DB.Create(&alarmRecord)
	if res.Error != nil {
		return res.Error
	}

	return nil
}

func (r *ActiveAlarmRepository) FindByNodeName(nodeName string) ([]ActiveAlarm, error) {
	var result []ActiveAlarm
	err := r.DB.Raw(`SELECT *
FROM 
    active_alarm
WHERE commit_id = ?
AND node_name = ?
`, r.CommitId, nodeName).Scan(&result).Error

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (r *ActiveAlarmRepository) UpdateSentTime(alarm ActiveAlarm, ts int64) error {

	err := r.DB.Exec(`
UPDATE active_alarm
SET alert_record_sent_time = ?
WHERE commit_id = ?
AND alarmer_name = ?
AND node_name = ?
AND strategy_target = ?
`, ts, r.CommitId, alarm.AlarmerName, alarm.NodeName, alarm.StrategyTarget).Error
	if err != nil {
		return err
	}

	return nil
}

func (r *ActiveAlarmRepository) DeleteListAll(alist []ActiveAlarm) error {
	if len(alist) == 0 {
		return nil
	}

	// Generate the conditions for a raw SQL query
	var params []interface{}
	query := "DELETE FROM active_alarm WHERE (node_name, strategy_target, alarmer_name) IN ("

	for i, alarm := range alist {
		if i > 0 {
			query += ","
		}
		query += "(?, ?, ?)"
		params = append(params, alarm.NodeName, alarm.StrategyTarget, alarm.AlarmerName)
	}

	query += ")"

	// Execute the raw SQL query
	if err := r.DB.Exec(query, params...).Error; err != nil {
		return fmt.Errorf("failed to delete alarms: %w", err)
	}

	return nil
}
