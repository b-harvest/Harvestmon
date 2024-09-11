package repository

import (
	"errors"
	"fmt"
)

type MetaMonitor struct {
	AgentName string `gorm:"primaryKey;column:agent_name;not null;type:varchar(50)"`
	Height    int64  `gorm:"column:height;not null;type:bigint"`
}

func (MetaMonitor) TableName() string {
	return "meta_monitor"
}

type MetaMonitorRepository struct {
	BaseRepository
}

func (r *MetaMonitorRepository) Save(metaMonitor MetaMonitor) error {
	res := r.DB.Save(&metaMonitor)
	if res.Error != nil {
		return res.Error
	}

	return nil
}

func (r *MetaMonitorRepository) FetchHighestHeight(agentName string) (uint64, error) {
	var (
		maxHeight uint64
	)
	err := r.DB.Raw(`select /*+ USE INDEX (INDEX_AGENT_NAME_HEIGHT) */ max(height)
from meta_monitor
where agent_name = ?
order by height desc
limit 1;`, agentName).Scan(&maxHeight).Error

	if err != nil {
		return 0, errors.New(fmt.Sprintf("failed to get maximum height: %v", err))
	}

	return maxHeight, nil
}
