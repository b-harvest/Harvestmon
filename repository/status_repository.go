package repository

import (
	"github.com/b-harvest/Harvestmon/log"
	"gorm.io/gorm/schema"
	"tendermint-mon/types"
	"time"
)

type TendermintNodeInfo struct {
	TendermintNodeInfoUUID string `gorm:"primaryKey;column:tendermint_node_info_uuid;not null;type:UUID"`

	NodeId     string `gorm:"column:node_id;not null;type:varchar(100)"`
	ListenAddr string `gorm:"column:listen_addr;not null;type:varchar(255)"`
	ChainId    string `gorm:"column:chain_id;not null;type:varchar(20)"`
	Moniker    string `gorm:"column:moniker;not null;type:varchar(50)"`

	TendermintPeerInfos []TendermintPeerInfo `gorm:"foreignKey:TendermintNodeInfoUUID;references:TendermintNodeInfoUUID"`
	TendermintNodeInfos []TendermintNodeInfo `gorm:"foreignKey:TendermintNodeInfoUUID;references:TendermintNodeInfoUUID"`
}

func (TendermintNodeInfo) TableName() string {
	return "tendermint_node_info"
}

type TendermintStatus struct {
	CreatedAt              time.Time          `gorm:"primaryKey;column:created_at;not null;type:datetime(6);autoCreateTime:false"`
	Event                  Event              `gorm:"foreignKey:EventUUID;references:EventUUID"`
	EventUUID              string             `gorm:"primaryKey;column:event_uuid;not null;type:UUID"`
	TendermintNodeInfo     TendermintNodeInfo `gorm:"foreignKey:TendermintNodeInfoUUID;references:TendermintNodeInfoUUID"`
	TendermintNodeInfoUUID string             `gorm:"column:tendermint_node_info_uuid;not null;type:UUID"`
	LatestBlockHash        string             `gorm:"column:latest_block_hash;not null;type:varchar(100)"`
	LatestAppHash          string             `gorm:"column:latest_app_hash;not null;type:varchar(100)"`
	LatestBlockHeight      uint64             `gorm:"column:latest_block_height;not null;type:bigint"`
	LatestBlockTime        time.Time          `gorm:"column:latest_block_time;not null;type:datetime(6)"`
	EarliestBlockHash      string             `gorm:"column:earliest_block_hash;not null;type:varchar(100)"`
	EarliestAppHash        string             `gorm:"column:earliest_app_hash;not null;type:varchar(100)"`
	EarliestBlockHeight    uint64             `gorm:"column:earliest_block_height;not null;type:bigint"`
	EarliestBlockTime      time.Time          `gorm:"column:earliest_block_time;not null;type:datetime(6)"`
	CatchingUp             bool               `gorm:"column:catching_up;not null;type:bool"`
}

func (TendermintStatus) TableName() string {
	return "tendermint_status"
}

type StatusRepository struct {
	BaseRepository
}

func (r *StatusRepository) Save(status TendermintStatus) error {
	eventAssociation := r.DB.Model(&status).Association("Event")
	eventAssociation.Relationship.Type = schema.BelongsTo
	err := eventAssociation.Append(&status.Event)
	if err != nil {
		return err
	}

	nodeInfoAssociation := r.DB.Model(&status).Association("TendermintNodeInfo")
	nodeInfoAssociation.Relationship.Type = schema.BelongsTo
	err = nodeInfoAssociation.Append(&status.TendermintNodeInfo)
	if err != nil {
		return err
	}

	res := r.DB.Create(&status)
	if res.Error != nil {
		return res.Error
	}

	log.Debug("Inserted into `tendermint_node_info`, `tendermint_status`, `event` successfully. eventUUID: " + status.Event.EventUUID)

	return nil
}

type TSEvent struct {
	AgentName         string    `gorm:"column:agent_name"`
	EventUUID         string    `gorm:"column:event_uuid"`
	CreatedAt         time.Time `gorm:"column:created_at;not null;type:datetime(6)"`
	LatestBlockHeight uint64    `gorm:"column:latest_block_height"`
	LatestBlockTime   time.Time `gorm:"column:latest_block_time;not null;type:datetime(6)"`
	CatchingUp        bool      `gorm:"column:catching_up;null"`
}

func (r *StatusRepository) FindTSEventsAfterStartTimeGroupByAgentName(startTime time.Time, agentName string) ([]TSEvent, error) {
	var result []TSEvent

	err := r.DB.Raw(`SELECT
    e.agent_name,
    ts.event_uuid,
    ts.created_at,
    ts.latest_block_height,
    ts.latest_block_time,
    ts.catching_up
FROM
    event e
        JOIN
    tendermint_status ts ON e.event_uuid = ts.event_uuid
WHERE e.created_at >= ?
    and e.service_name = ?
    and e.event_type = 'tm:event:status'
  and e.agent_name = ?
and e.commit_id = ?
ORDER BY e.agent_name,ts.created_at DESC;
`, types.HARVEST_SERVICE_NAME, startTime, agentName, r.CommitId).Scan(&result).Error
	if err != nil {
		return nil, err
	}

	return result, nil
}
