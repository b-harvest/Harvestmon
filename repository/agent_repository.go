package repository

import (
	"errors"
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
	MarkerUsername     string     `gorm:"column:marker_username;not null;type:varchar(100)"`
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

func (r *AgentMarkRepository) Save(mark AgentMark) error {
	var existingMark AgentMark
	res := r.DB.Where("mark_start >= ? and mark_start < ?", mark.MarkStart.Add(-(1 * time.Minute)), mark.MarkStart).First(&existingMark)

	if res.Error != nil && !errors.Is(res.Error, gorm.ErrRecordNotFound) {
		return res.Error
	}

	if errors.Is(res.Error, gorm.ErrRecordNotFound) {
		createRes := r.DB.Create(&mark)
		if createRes.Error != nil {
			return createRes.Error
		}

		log.Debug("Inserted `agent_mark`")
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
