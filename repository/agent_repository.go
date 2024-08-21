package repository

import (
	"errors"
	"fmt"
	"github.com/b-harvest/Harvestmon/log"
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

type AgentRepository struct {
	BaseRepository
}

func (r *AgentRepository) FindAgentByAgentName(agentName string) (*Agent, error) {
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

func (r *AgentRepository) FindAll() ([]Agent, error) {
	var result []Agent

	err := r.DB.Raw(`select * 
from agent
where commit_id = ?`, r.CommitId).Scan(&result).Error

	if err != nil {
		return nil, err
	}

	return result, nil
}

type AgentMarkRepository struct {
	BaseRepository
}

func (r *AgentMarkRepository) Delete(mark AgentMark) error {
	if err := r.DB.Where("agent_name = ? AND mark_start = ?", mark.AgentName, mark.MarkStart).Delete(&AgentMark{}).Error; err != nil {
		return errors.New("Failed to delete record: " + err.Error())
	} else {
		log.Debug(fmt.Sprintf("Deleted record(s) for AgentName '%s' with specified MarkStart", mark.AgentName))
		return nil
	}
}

func (r *AgentMarkRepository) Save(mark AgentMark) error {
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
		log.Debug("Created new `agent_mark`")
	} else {
		// Record exists, so update it
		updateRes := r.DB.Model(&existingMark).Updates(mark)
		if updateRes.Error != nil {
			// If there is an error during update, return it
			return updateRes.Error
		}
		log.Debug("Updated existing `agent_mark`")
	}

	return nil
}

func (r *AgentMarkRepository) FindAgentMarkByAgentNameAndTime(agentName string, time time.Time) (*AgentMark, error) {
	var result AgentMark

	err := r.DB.Raw(`select *
from agent_mark
where agent_name = ?
and (mark_end is null 
or mark_end >= ?)`, agentName, time).Scan(&result).Error

	if err != nil {
		return nil, err
	}

	return &result, nil

}
