package repository

import (
	"errors"
	"fmt"
	"gorm.io/gorm"
	"time"
)

type Agent struct {
	AgentName string `gorm:"column:agent_name;not null;type:varchar(100)"`
	CommitID  string `gorm:"column:commit_id;not null;type:varchar(255)"`
	Host      string `gorm:"column:host;not null;type:varchar(30)"`
	Port      int    `gorm:"column:port;null;type:int"`
	Platform  string `gorm:"column:platform;null;type:varchar(255)"`
	Location  string `gorm:"column:location;null;type:varchar(255)"`
}

func (Agent) TableName() string {
	return "agent"
}

type AgentMark struct {
	AgentName          string     `gorm:"column:agent_name;not null;type:varchar(100)"`
	MarkStart          *time.Time `gorm:"column:mark_start;not null;type:datetime(6);autoCreateTime:false"`
	MarkEnd            *time.Time `gorm:"column:mark_end;null;type:datetime(6);autoCreateTime:false"`
	MarkerUserIdentity string     `gorm:"column:marker_user_identity;not null;type:varchar(255)"`
	MarkerFrom         string     `gorm:"column:marker_from;not null;type:varchar(255)"`
}

func (AgentMark) TableName() string {
	return "agent_mark"
}

func (r *Repository) FindAgentByAgentName(agentName string) (*Agent, error) {
	var result Agent

	err := r.DB.Raw(`select * 
from agent
where agent_name = ?
and commit_id = ?`, agentName, r.CommitId).Scan(&result).Error

	if err != nil {
		return nil, err
	}

	if result.AgentName == "" {
		return nil, errors.New("agent not found")
	}

	return &result, nil
}

func (r *Repository) FindAgentsAll() ([]Agent, error) {
	var result []Agent

	err := r.DB.Raw(`select * 
from agent
where commit_id = ?`, r.CommitId).Scan(&result).Error

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (r *Repository) DeleteAgentMark(mark AgentMark) error {
	if err := r.DB.Where("agent_name = ? AND mark_start = ?", mark.AgentName, mark.MarkStart).Delete(&AgentMark{}).Error; err != nil {
		return errors.New("Failed to delete record: " + err.Error())
	} else {
		return nil
	}
}

func (r *Repository) DeleteAgentMarks(marks []AgentMark) error {
	if len(marks) == 0 {
		return nil
	}

	// Generate the conditions for a raw SQL query
	var params []interface{}
	query := "DELETE FROM agent_mark WHERE (agent_name, mark_start) IN ("

	for i, mark := range marks {
		if i > 0 {
			query += ","
		}
		query += "(?, ?)"
		params = append(params, mark.AgentName, mark.MarkStart)
	}

	query += ")"

	// Execute the raw SQL query
	if err := r.DB.Exec(query, params...).Error; err != nil {
		return fmt.Errorf("failed to delete alarms: %w", err)
	}
	return nil
}

func (r *Repository) SaveOrUpdateAgentMark(mark AgentMark) error {
	var existingMark AgentMark

	// Check if a record already exists with the specified conditions
	findRes := r.DB.Where("agent_name = ? AND mark_start = ? AND marker_user_identity = ?",
		mark.AgentName, mark.MarkStart, mark.MarkerUserIdentity).First(&existingMark)

	if findRes.Error != nil && !errors.Is(findRes.Error, gorm.ErrRecordNotFound) {
		// If there's an error that's not "record not found", return the error
		return findRes.Error
	}

	if errors.Is(findRes.Error, gorm.ErrRecordNotFound) {
		// Record does not exist, so create a new one
		createRes := r.DB.Create(&mark)
		if createRes.Error != nil {
			// If there is an error during creation, return it
			return createRes.Error
		}
	} else {
		// Record exists, so update it
		updateRes := findRes.Model(&existingMark).Updates(mark)
		if updateRes.Error != nil {
			// If there is an error during update, return it
			return updateRes.Error
		}
	}

	return nil
}

func (r *Repository) FindAgentMarkByAgentNameAndTime(agentName string, time time.Time) ([]AgentMark, error) {
	var result []AgentMark

	err := r.DB.Raw(`select *
from agent_mark
where agent_name = ?
and (mark_end is null 
or mark_end >= ?)`, agentName, time).Scan(&result).Error

	if err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return []AgentMark{}, nil
	}

	return result, nil
}

func (r *Repository) FindAgentMarkByAgentNameLimit(limit int) ([]AgentMark, error) {
	var result []AgentMark

	err := r.DB.Raw(`
		select *
		from agent_mark
		order by mark_start desc
		limit ?`, limit).Scan(&result).Error

	if err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return []AgentMark{}, nil
	}

	return result, nil
}
