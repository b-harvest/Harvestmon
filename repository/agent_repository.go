package repository

import (
	"errors"
	"gorm.io/gorm"
	"strings"
	"time"
)

type Agent struct {
	Instance    string  `gorm:"primaryKey;column:instance;not null;type:varchar(100)"`
	Target      string  `gorm:"column:target;not null;type:varchar(30)"`
	MetricsPath string  `gorm:"column:metrics_path;null;type:varchar(255)"`
	Scheme      string  `gorm:"column:scheme;null;type:varchar(255)"`
	Labels      []Label `gorm:"-"`
	LabelKeys   string  `gorm:"column:label_keys;type:text"` // Store comma-separated Label composite keys
}

func (Agent) TableName() string {
	return "agent"
}

type Label struct {
	Key       string  `gorm:"primaryKey;column:label_key;not null;type:varchar(100)"`
	Value     string  `gorm:"primaryKey;column:value;not null;type:varchar(255)"`
	Agents    []Agent `gorm:"-"`
	AgentKeys string  `gorm:"column:agent_keys;type:text"` // Store comma-separated Agent keys
}

func (Label) TableName() string {
	return "label"
}

// Label BeforeSave: Serialize Agents to agent_keys before saving
func (l *Label) BeforeSave(tx *gorm.DB) (err error) {
	var keys []string
	for _, agent := range l.Agents {
		keys = append(keys, agent.Instance)
	}
	l.AgentKeys = strings.Join(keys, ",")
	return nil
}

// Label AfterFind: Deserialize agent_keys into Agents after loading
func (l *Label) AfterFind(tx *gorm.DB) (err error) {
	if l.AgentKeys == "" {
		return nil
	}

	keys := strings.Split(l.AgentKeys, ",")
	var agents []Agent
	if len(keys) > 0 {
		tx.Where("instance IN ?", keys).Find(&agents)
	}
	l.Agents = agents
	return nil
}

// Agent BeforeSave: Serialize Labels to label_keys before saving
func (a *Agent) BeforeSave(tx *gorm.DB) (err error) {
	var keys []string
	for _, label := range a.Labels {
		// Combine Key and Value as the composite key representation
		keys = append(keys, label.Key+"|"+label.Value)
	}
	a.LabelKeys = strings.Join(keys, ",")
	return nil
}

// Agent AfterFind: Deserialize label_keys into Labels after loading
func (a *Agent) AfterFind(tx *gorm.DB) (err error) {
	if a.LabelKeys == "" {
		return nil
	}

	compositeKeys := strings.Split(a.LabelKeys, ",")
	var labels []Label
	for _, compositeKey := range compositeKeys {
		parts := strings.SplitN(compositeKey, "|", 2)
		if len(parts) == 2 {
			labels = append(labels, Label{Key: parts[0], Value: parts[1]})
		}
	}
	if len(labels) > 0 {
		tx.Find(&labels)
	}
	a.Labels = labels
	return nil
}

type AgentMark struct {
	AgentName          string     `gorm:"primaryKey;column:agent_name;not null;type:varchar(100)"`
	MarkStart          *time.Time `gorm:"primaryKey;column:mark_start;not null;type:datetime(6);autoCreateTime:false"`
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
where agent_name = ?`, agentName).Scan(&result).Error

	if err != nil {
		return nil, err
	}

	if result.Instance == "" {
		return nil, errors.New("agent not found")
	}

	return &result, nil
}

func (r *Repository) FindAgents() ([]Agent, error) {
	var result []Agent

	err := r.DB.Raw(`select * 
from agent`).Scan(&result).Error

	if err != nil {
		return nil, err
	}

	return result, nil
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
