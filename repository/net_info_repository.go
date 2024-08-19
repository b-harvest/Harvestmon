package repository

import (
	log "github.com/b-harvest/Harvestmon/log"
	"gorm.io/gorm/schema"
	"time"
)

type TendermintNetInfo struct {
	CreatedAt           time.Time            `gorm:"primaryKey;column:created_at;not null;type:datetime(6);autoCreateTime:false"`
	Event               Event                `gorm:"foreignKey:EventUUID;references:EventUUID"`
	EventUUID           string               `gorm:"primaryKey;column:event_uuid;not null;type:CHAR(36)"`
	NPeers              int                  `gorm:"column:n_peers;not null;type:int"`
	Listening           bool                 `gorm:"column:listening;not null;type:bool"`
	TendermintPeerInfos []TendermintPeerInfo `gorm:"foreignKey:TendermintNetInfoCreatedAt;references:CreatedAt"`
}

func (TendermintNetInfo) TableName() string {
	return "tendermint_net_info"
}

type TendermintPeerInfo struct {
	TendermintPeerInfoUUID     string             `gorm:"column:tendermint_peer_info_uuid;not null;type:CHAR(36)"`
	TendermintNetInfoCreatedAt time.Time          `gorm:"column:created_at;not null;type:datetime(6);autoCreateTime:false"`
	Event                      Event              `gorm:"foreignKey:EventUUID;references:EventUUID"`
	EventUUID                  string             `gorm:"column:event_uuid;not null;type:CHAR(36)"`
	IsOutbound                 bool               `gorm:"column:is_outbound;not null;type:bool"`
	TendermintNodeInfo         TendermintNodeInfo `gorm:"foreignKey:TendermintNodeInfoUUID;references:TendermintNodeInfoUUID"`
	TendermintNodeInfoUUID     string             `gorm:"column:tendermint_node_info_uuid;not null;type:CHAR(36)"`
	RemoteIP                   string             `gorm:"column:remote_ip;not null;type:varchar(50)"`
}

func (TendermintPeerInfo) TableName() string {
	return "tendermint_peer_info"
}

type NetInfoRepository struct {
	BaseRepository
}

func (r *NetInfoRepository) Save(netInfo TendermintNetInfo) error {
	// Insert event
	//err := r.EventRepository.Save(event)
	//if err != nil {
	//	return err
	//}

	// Insert tendermint_node_info
	//res, err := r.DB.ExecContext(context.Background(), "INSERT INTO harvestmon.tendermint_net_info (created_at, event_uuid, n_peers, listening) VALUES (?, ?, ?, ?)",
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

	eventAssociation := r.DB.Model(&netInfo).Association("Event")
	eventAssociation.Relationship.Type = schema.BelongsTo
	err := eventAssociation.Append(&netInfo.Event)
	if err != nil {
		return err
	}

	for _, peerInfo := range netInfo.TendermintPeerInfos {
		nodeInfoAssociation := r.DB.Model(&peerInfo).Association("TendermintNodeInfo")
		nodeInfoAssociation.Relationship.Type = schema.BelongsTo
		err = nodeInfoAssociation.Append(&peerInfo.TendermintNodeInfo)
		if err != nil {
			return err
		}
	}

	res := r.DB.Create(&netInfo)
	if res.Error != nil {
		return res.Error
	}

	//for _, peer := range peerInfos {
	//	// Insert tendermint_node_info
	//	//res, err = r.DB.ExecContext(context.Background(), "INSERT INTO harvestmon.tendermint_node_info (tendermint_node_info_uuid, node_id, listen_addr, chain_id, moniker) values (?, ?, ?, ?, ?)",
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
	//	res = r.DB.Create(&peer)
	//
	//	if res.Error != nil {
	//		return res.Error
	//	}
	//}

	log.Debug("Inserted `tendermint_net_info`, `tendermint_peer_info`, `tendermint_node_info`, `event` successfully. eventUUID: " + netInfo.Event.EventUUID)

	return nil
}

type AgentPeerInfo struct {
	AgentName         string    `gorm:"column:agent_name"`
	EventUUID         string    `gorm:"column:event_uuid"`
	CreatedAt         time.Time `gorm:"column:created_at;not null;type:datetime(6)"`
	NPeers            int       `gorm:"column:n_peers"`
	PeerInfoUUIDCount int       `gorm:"column:tpi_count"`
}

func (r *NetInfoRepository) FindLatestAgentPeerInfosByAgentName(agentName, eventType, serviceName string) ([]AgentPeerInfo, error) {
	var result []AgentPeerInfo

	err := r.DB.Raw(`SELECT 
    e.agent_name, 
    e.event_uuid, 
    tni.created_at, 
    tni.n_peers, 
    COUNT(tpi.tendermint_peer_info_uuid) AS tpi_count
FROM 
    event e
JOIN 
    tendermint_net_info tni 
    ON e.event_uuid = tni.event_uuid
JOIN 
    tendermint_peer_info tpi 
    ON tni.event_uuid = tpi.event_uuid 
    AND tni.created_at = tpi.created_at
JOIN (
    SELECT 
        agent_name, 
        MAX(created_at) AS max_created_at
    FROM 
        event
    WHERE 
        event_type = ?
      and agent_name = ?
        AND service_name = ?
    GROUP BY 
        agent_name
) max_ein 
ON e.agent_name = max_ein.agent_name 
AND e.created_at = max_ein.max_created_at
WHERE 
    e.event_type = ?
    AND e.service_name = ?
and e.commit_id = ?
GROUP BY 
    e.agent_name, e.event_uuid, tni.created_at, tni.n_peers;
`, eventType, agentName, serviceName, eventType, serviceName, r.CommitId).Scan(&result).Error

	if err != nil {
		return nil, err
	}

	return result, nil
}
