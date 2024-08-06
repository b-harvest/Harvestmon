package repository

import (
	"github.com/b-harvest/Harvestmon/log"
	"gorm.io/gorm/schema"
	"time"
)

type TendermintNodeInfo struct {
	TendermintNodeInfoUUID string `gorm:"column:tendermint_node_info_uuid;not null;type:UUID"`

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
	CreatedAt              time.Time          `gorm:"column:created_at;not null;type:datetime(6);autoCreateTime:false"`
	Event                  Event              `gorm:"foreignKey:EventUUID;references:EventUUID"`
	EventUUID              string             `gorm:"column:event_uuid;not null;type:UUID"`
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
	EventRepository
}

func (r *StatusRepository) Save(status TendermintStatus) error {
	// Insert event
	//err := r.EventRepository.Save(event)
	//if err != nil {
	//	return err
	//}

	// Insert tendermint_node_info
	//res, err := r.Db.ExecContext(context.Background(), "INSERT INTO harvestmon.tendermint_node_info (tendermint_node_info_uuid, node_id, listen_addr, chain_id, moniker) values (?, ?, ?, ?, ?)",
	//	nodeInfo.TendermintNodeInfoUUID, string(nodeInfo.NodeId),
	//	nodeInfo.ListenAddr,
	//	nodeInfo.ChainId, nodeInfo.Moniker)
	//if err != nil {
	//	return err
	//}
	//
	//_, err = res.RowsAffected()
	//if err != nil {
	//	return err
	//}
	//res := r.Db.Create(&nodeInfo)
	//if res.Error != nil {
	//	return res.Error
	//}

	// Insert tendermint_status

	//res, err = r.Db.ExecContext(context.Background(), "INSERT INTO harvestmon.tendermint_status (created_at, event_uuid, tendermint_node_info_uuid, latest_block_hash, latest_app_hash, latest_block_height, latest_block_time, earliest_block_hash, earliest_app_hash, earliest_block_height, earliest_block_time, catching_up) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
	//	status.CreatedAt,
	//	status.EventUUID,
	//	status.TendermintNodeInfoUUID,
	//	status.LatestBlockHash,
	//	status.LatestAppHash,
	//	status.LatestBlockHeight,
	//	status.LatestBlockTime,
	//	status.EarliestBlockHash,
	//	status.EarliestAppHash,
	//	status.EarliestBlockHeight,
	//	status.EarliestBlockTime,
	//	status.CatchingUp,
	//)
	//if err != nil {
	//	return err
	//}
	//
	//_, err = res.RowsAffected()
	//if err != nil {
	//	return err
	//}

	eventAssociation := r.Db.Model(&status).Association("Event")
	eventAssociation.Relationship.Type = schema.BelongsTo
	err := eventAssociation.Append(&status.Event)
	if err != nil {
		return err
	}

	nodeInfoAssociation := r.Db.Model(&status).Association("TendermintNodeInfo")
	nodeInfoAssociation.Relationship.Type = schema.BelongsTo
	err = nodeInfoAssociation.Append(&status.TendermintNodeInfo)
	if err != nil {
		return err
	}

	res := r.Db.Create(&status)
	if res.Error != nil {
		return res.Error
	}

	log.Debug("Inserted into `tendermint_node_info`, `tendermint_status`, `event` successfully. eventUUID: " + status.Event.EventUUID)

	return nil
}

type TendermintStatusAndEvent struct {
	TendermintStatus
	Event
}

func (r *StatusRepository) FindStatusAndEventByServiceNameOrderByCreatedAtDescWithLimitGroupByAgentName(serviceName string, limit int) []TendermintStatusAndEvent {
	// Fetch where is_checked = false
	return nil
}
