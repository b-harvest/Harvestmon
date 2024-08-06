package repository

import (
	log "github.com/b-harvest/Harvestmon/log"
	"gorm.io/gorm/schema"
	"time"
)

type TendermintNetInfo struct {
	CreatedAt           time.Time            `gorm:"column:created_at;not null;type:datetime(6);autoCreateTime:false"`
	Event               Event                `gorm:"foreignKey:EventUUID;references:EventUUID"`
	EventUUID           string               `gorm:"column:event_uuid;not null;type:UUID"`
	NPeers              int                  `gorm:"column:n_peers;not null;type:int"`
	Listening           bool                 `gorm:"column:listening;not null;type:bool"`
	TendermintPeerInfos []TendermintPeerInfo `gorm:"foreignKey:TendermintNetInfoCreatedAt;references:CreatedAt"`
}

func (TendermintNetInfo) TableName() string {
	return "tendermint_net_info"
}

type TendermintPeerInfo struct {
	TendermintPeerInfoUUID     string             `gorm:"column:tendermint_peer_info_uuid;not null;type:UUID"`
	TendermintNetInfoCreatedAt time.Time          `gorm:"column:created_at;not null;type:datetime(6);autoCreateTime:false"`
	Event                      Event              `gorm:"foreignKey:EventUUID;references:EventUUID"`
	EventUUID                  string             `gorm:"column:event_uuid;not null;type:UUID"`
	IsOutbound                 bool               `gorm:"column:is_outbound;not null;type:bool"`
	TendermintNodeInfo         TendermintNodeInfo `gorm:"foreignKey:TendermintNodeInfoUUID;references:TendermintNodeInfoUUID"`
	TendermintNodeInfoUUID     string             `gorm:"column:tendermint_node_info_uuid;not null;type:UUID"`
	RemoteIP                   string             `gorm:"column:remote_ip;not null;type:varchar(50)"`
}

func (TendermintPeerInfo) TableName() string {
	return "tendermint_peer_info"
}

type NetInfoRepository struct {
	EventRepository
}

func (r *NetInfoRepository) Save(netInfo TendermintNetInfo) error {
	// Insert event
	//err := r.EventRepository.Save(event)
	//if err != nil {
	//	return err
	//}

	// Insert tendermint_node_info
	//res, err := r.Db.ExecContext(context.Background(), "INSERT INTO harvestmon.tendermint_net_info (created_at, event_uuid, n_peers, listening) VALUES (?, ?, ?, ?)",
	//	netInfo.CreatedAt, netInfo.EventUUID,
	//	netInfo.NPeers,
	//	netInfo.Listening)
	//if err != nil {
	//	return err
	//}
	//
	//_, err = res.RowsAffected()
	//if err != nil {
	//	return err
	//}

	eventAssociation := r.Db.Model(&netInfo).Association("Event")
	eventAssociation.Relationship.Type = schema.BelongsTo
	err := eventAssociation.Append(&netInfo.Event)
	if err != nil {
		return err
	}

	for _, peerInfo := range netInfo.TendermintPeerInfos {
		nodeInfoAssociation := r.Db.Model(&peerInfo).Association("TendermintNodeInfo")
		nodeInfoAssociation.Relationship.Type = schema.BelongsTo
		err = nodeInfoAssociation.Append(&peerInfo.TendermintNodeInfo)
		if err != nil {
			return err
		}
	}

	res := r.Db.Create(&netInfo)
	if res.Error != nil {
		return res.Error
	}

	//for _, peer := range peerInfos {
	//	// Insert tendermint_node_info
	//	//res, err = r.Db.ExecContext(context.Background(), "INSERT INTO harvestmon.tendermint_node_info (tendermint_node_info_uuid, node_id, listen_addr, chain_id, moniker) values (?, ?, ?, ?, ?)",
	//	//	peer.TendermintNodeInfo.TendermintNodeInfoUUID, string(peer.TendermintNodeInfo.NodeId),
	//	//	peer.TendermintNodeInfo.ListenAddr,
	//	//	peer.TendermintNodeInfo.ChainId, peer.TendermintNodeInfo.Moniker)
	//	//if err != nil {
	//	//	return err
	//	//}
	//	//
	//	//_, err = res.RowsAffected()
	//	//if err != nil {
	//	//	return err
	//	//}
	//	res = r.Db.Create(&peer)
	//
	//	if res.Error != nil {
	//		return res.Error
	//	}
	//}

	log.Debug("Inserted `tendermint_net_info`, `tendermint_peer_info`, `tendermint_node_info`, `event` successfully. eventUUID: " + netInfo.Event.EventUUID)

	return nil
}

type TendermintNetInfoAndPeerInfos struct {
	TendermintNetInfo
	PeerInfos []TendermintPeerInfo
	Event
}

func (r *NetInfoRepository) FindNetInfoAndPeerInfosAndEventByServiceNameOrderByCreatedAtDescWithLimitGroupByAgentName(serviceName string, limit int) []TendermintNetInfoAndPeerInfos {
	return nil
}
