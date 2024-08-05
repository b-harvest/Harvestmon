package repository

import (
	"context"
	"github.com/adlio/schema"
	log "github.com/b-harvest/Harvestmon/harvestmon-log"
	"tendermint-mon/types"
	"time"
)

type TendermintNetInfo struct {
	CreatedAt time.Time
	EventUUID string
	NPeers    int
	Listening bool
}

type TendermintPeerInfo struct {
	TendermintPeerInfoUUID string
	CreatedAt              time.Time
	EventUUID              string
	IsOutbound             bool
	TendermintNodeInfo     TendermintNodeInfo
	RemoteIP               string
}

type NetInfoMonitorRepository struct {
	Db        schema.Queryer
	EventType string
	Agent     types.MonitoringAgent
}

func (r *NetInfoMonitorRepository) Save(event Event, netInfo TendermintNetInfo, peerInfos []TendermintPeerInfo) error {
	// Insert event
	res, err := r.Db.ExecContext(context.Background(), "INSERT INTO harvestmon.event (event_uuid, agent_name, service_name, commit_id, event_type, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		event.EventUUID, event.AgentName, event.ServiceName, event.CommitID, event.EventType, event.CreatedAt)
	if err != nil {
		return err
	}

	_, err = res.RowsAffected()
	if err != nil {
		return err
	}

	// Insert tendermint_node_info
	res, err = r.Db.ExecContext(context.Background(), "INSERT INTO harvestmon.tendermint_net_info (created_at, event_uuid, n_peers, listening) VALUES (?, ?, ?, ?)",
		netInfo.CreatedAt, netInfo.EventUUID,
		netInfo.NPeers,
		netInfo.Listening)
	if err != nil {
		return err
	}

	_, err = res.RowsAffected()
	if err != nil {
		return err
	}

	for _, peer := range peerInfos {
		// Insert tendermint_node_info
		res, err = r.Db.ExecContext(context.Background(), "INSERT INTO harvestmon.tendermint_node_info (tendermint_node_info_uuid, node_id, listen_addr, chain_id, moniker) values (?, ?, ?, ?, ?)",
			peer.TendermintNodeInfo.TendermintNodeInfoUUID, string(peer.TendermintNodeInfo.NodeId),
			peer.TendermintNodeInfo.ListenAddr,
			peer.TendermintNodeInfo.ChainId, peer.TendermintNodeInfo.Moniker)
		if err != nil {
			return err
		}

		_, err = res.RowsAffected()
		if err != nil {
			return err
		}

		// Insert tendermint_status
		res, err = r.Db.ExecContext(context.Background(), "INSERT INTO harvestmon.tendermint_peer_info (tendermint_peer_info_uuid, created_at, event_uuid, is_outbound, tendermint_node_info_uuid, remote_ip) VALUES(?, ?, ?, ?, ?, ?)",
			peer.TendermintPeerInfoUUID,
			peer.CreatedAt,
			peer.EventUUID,
			peer.IsOutbound,
			peer.TendermintNodeInfo.TendermintNodeInfoUUID,
			peer.RemoteIP)
		if err != nil {
			return err
		}

		_, err = res.RowsAffected()
		if err != nil {
			return err
		}
	}

	log.Debug("Inserted `tendermint_net_info`, `tendermint_peer_info`, `tendermint_node_info`, `event` successfully. eventUUID: " + event.EventUUID)

	return nil
}
