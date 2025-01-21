package repository

import (
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

type Agent struct {
	Instance   string      `gorm:"primaryKey;column:instance;not null;type:varchar(100)"`
	Labels     []Label     `gorm:"-"` // Related Labels
	AgentMarks []AgentMark `gorm:"foreignKey:Instance;references:Instance"`
}

func (Agent) TableName() string {
	return "agent"
}

func (a *Agent) AfterSave(tx *gorm.DB) error {
	return tx.Transaction(func(tx *gorm.DB) error {
		// Step 1: Save related Labels
		for _, label := range a.Labels {
			// Use Upsert or a similar approach to avoid duplicate entry errors
			if err := tx.Clauses(clause.OnConflict{
				DoNothing: true, // Skip the insert if the record already exists
			}).Create(&label).Error; err != nil {
				return err
			}
		}

		// Step 2: Save Agent-Label associations
		if len(a.Labels) > 0 {
			// Clear existing associations in the join table
			if err := tx.Where("agent_instance = ?", a.Instance).Delete(&AgentLabel{}).Error; err != nil {
				return err
			}

			// Add new associations to the join table
			for _, label := range a.Labels {
				association := AgentLabel{
					AgentInstance: a.Instance,
					LabelKey:      label.Key,
					LabelValue:    label.Value,
				}
				if err := tx.Create(&association).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (a *Agent) AfterFind(tx *gorm.DB) error {
	// Load related Labels
	if tx != nil {
		var labels []Label
		err := tx.Raw(`
            SELECT l.* 
            FROM label l
            INNER JOIN agent_labels al 
            ON l.label_key = al.label_key AND l.value = al.value
            WHERE al.agent_instance = ?`, a.Instance).Scan(&labels).Error
		if err != nil {
			return err
		}
		a.Labels = labels
	}

	return nil
}

func (a *Agent) BeforeDelete(tx *gorm.DB) error {
	// Delete associations in the join table
	if err := tx.Where("agent_instance = ?", a.Instance).Delete(&AgentLabel{}).Error; err != nil {
		return err
	}

	return nil
}

// Label model definition
type Label struct {
	Key    string  `gorm:"primaryKey;column:label_key;not null;type:varchar(100)"`
	Value  string  `gorm:"primaryKey;column:value;not null;type:varchar(255)"`
	Agents []Agent `gorm:"-"` // Related Agents
}

// TableName returns the table name for Label
func (Label) TableName() string {
	return "label"
}

// AfterFind: Load related Agents after fetching
func (l *Label) AfterFind(tx *gorm.DB) (err error) {
	if tx != nil {
		var agents []Agent
		err = tx.Raw(`
            SELECT a.* FROM agent a
            INNER JOIN agent_labels al ON a.instance = al.agent_instance
            WHERE al.label_key = ? AND al.value = ?`, l.Key, l.Value).Scan(&agents).Error
		l.Agents = agents
	}
	return
}

// BeforeDelete: Remove associations before deleting
func (l *Label) BeforeDelete(tx *gorm.DB) (err error) {
	return tx.Where("label_key = ? AND value = ?", l.Key, l.Value).Delete(&AgentLabel{}).Error
}

// AgentLabel join table definition
type AgentLabel struct {
	AgentInstance string    `gorm:"primaryKey;column:agent_instance;not null;type:varchar(100)"`
	LabelKey      string    `gorm:"primaryKey;column:label_key;not null;type:varchar(100)"`
	LabelValue    string    `gorm:"primaryKey;column:value;not null;type:varchar(255)"`
	CreatedAt     time.Time `gorm:"column:created_at"`
}

// TableName returns the table name for AgentLabel
func (AgentLabel) TableName() string {
	return "agent_labels"
}

type AgentMark struct {
	MarkStart          *time.Time `gorm:"primaryKey;column:mark_start;not null;type:datetime(6);autoCreateTime:false"`
	MarkEnd            *time.Time `gorm:"column:mark_end;null;type:datetime(6);autoCreateTime:false"`
	MarkerUserIdentity string     `gorm:"column:marker_user_identity;not null;type:varchar(255)"`
	MarkerFrom         string     `gorm:"column:marker_from;not null;type:varchar(255)"`
	Instance           string     `gorm:"column:instance;not null;type:varchar(100)"`
}

func (AgentMark) TableName() string {
	return "agent_mark"
}

func (r *Repository) FindAgentByInstance(instanceName string) (*Agent, error) {
	var result Agent

	err := r.DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Raw(`select * 
from agent
where instance = ?`, instanceName).Scan(&result).Error
		if err != nil {
			return err
		}

		return result.AfterFind(tx)
	})

	if err != nil {
		return nil, err
	}

	if result.Instance == "" {
		return nil, errors.New("agent not found")
	}

	return &result, nil
}

func (r *Repository) FindAgentByLabel(labels map[string]string) ([]*Agent, error) {
	var (
		result []*Agent
		err    error
	)

	err = r.DB.Transaction(func(tx *gorm.DB) error {
		var instances = make(map[string]int)
		for k, v := range labels {
			var agentLabel AgentLabel
			err = tx.Raw(`select * 
from agent_labels
where label_key = ?
and value = ?`, k, v).Scan(&agentLabel).Error
			if err != nil {
				return err
			}
			instances[agentLabel.AgentInstance]++
		}

		for instance, size := range instances {
			if size == len(labels) {
				var agent Agent
				err = tx.Raw(`select *
from agent
where instance = ?`, instance).Scan(&agent).Error
				if err != nil {
					return err
				}
				result = append(result, &agent)
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (r *Repository) FindAgents() ([]*Agent, error) {
	var result []*Agent

	err := r.DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Raw(`select * 
from agent`).Scan(&result).Error
		if err != nil {
			return err
		}
		for _, agent := range result {
			err = agent.AfterFind(tx)
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (r *Repository) FindAgentMarkByInstanceAndTime(instance string, time *time.Time) ([]*AgentMark, error) {
	var result []*AgentMark

	err := r.DB.Transaction(func(tx *gorm.DB) error {
		if time == nil {
			err := tx.Raw(`select * 
from agent_mark
where instance = ? 
and mark_end is null`, instance).Scan(&result).Error
			if err != nil {
				return err
			}
		} else {
			err := tx.Raw(`select * 
from agent_mark
where instance = ?
and mark_start < ? 
and mark_end > ?`, instance, time, time).Scan(&result).Error
			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}
