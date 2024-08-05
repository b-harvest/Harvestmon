package repository

import (
	"context"
	"github.com/adlio/schema"
	log "github.com/b-harvest/harvestmon-log"
	"tendermint-mon/types"
	"time"
)

type TendermintNodeInfo struct {
	TendermintNodeInfoUUID string
	NodeId                 string
	ListenAddr             string
	ChainId                string
	Moniker                string
}

type TendermintStatus struct {
	CreatedAt              time.Time
	EventUUID              string
	TendermintNodeInfoUUID string
	LatestBlockHash        string
	LatestAppHash          string
	LatestBlockHeight      uint64
	LatestBlockTime        time.Time
	EarliestBlockHash      string
	EarliestAppHash        string
	EarliestBlockHeight    uint64
	EarliestBlockTime      time.Time
	CatchingUp             bool
}

type StatusMonitorRepository struct {
	Db        schema.Queryer
	EventType string
	Agent     types.MonitoringAgent
}

func (r *StatusMonitorRepository) Save(event Event, nodeInfo TendermintNodeInfo, status TendermintStatus) error {
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
	res, err = r.Db.ExecContext(context.Background(), "INSERT INTO harvestmon.tendermint_node_info (tendermint_node_info_uuid, node_id, listen_addr, chain_id, moniker) values (?, ?, ?, ?, ?)",
		nodeInfo.TendermintNodeInfoUUID, string(nodeInfo.NodeId),
		nodeInfo.ListenAddr,
		nodeInfo.ChainId, nodeInfo.Moniker)
	if err != nil {
		return err
	}

	_, err = res.RowsAffected()
	if err != nil {
		return err
	}

	// Insert tendermint_status

	res, err = r.Db.ExecContext(context.Background(), "INSERT INTO harvestmon.tendermint_status (created_at, event_uuid, tendermint_node_info_uuid, latest_block_hash, latest_app_hash, latest_block_height, latest_block_time, earlist_block_hash, earlist_app_hash, earlist_block_height, earlist_block_time, catching_up) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		status.CreatedAt,
		status.EventUUID,
		status.TendermintNodeInfoUUID,
		status.LatestBlockHash,
		status.LatestAppHash,
		status.LatestBlockHeight,
		status.LatestBlockTime,
		status.EarliestBlockHash,
		status.EarliestAppHash,
		status.EarliestBlockHeight,
		status.EarliestBlockTime,
		status.CatchingUp,
	)
	if err != nil {
		return err
	}

	_, err = res.RowsAffected()
	if err != nil {
		return err
	}

	log.Debug("Inserted into `tendermint_node_info`, `tendermint_status`, `event` successfully. eventUUID: " + event.EventUUID)

	return nil
}
