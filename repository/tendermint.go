package repository

import (
	"errors"
	"fmt"
	"gorm.io/gorm"
	"time"
)

type TendermintCommit struct {
	CreatedAt          time.Time                   `gorm:"primaryKey;column:created_at;not null;type:datetime(6)"`
	Event              Event                       `gorm:"foreignKey:EventUUID;references:EventUUID"`
	EventUUID          string                      `gorm:"primaryKey;column:event_uuid;not null;type:CHAR(36)"`
	ChainID            string                      `gorm:"column:chain_id;not null;type:varchar(20)"`
	Height             string                      `gorm:"column:height;not null;type:bigint"`
	Time               time.Time                   `gorm:"column:time;not null;type:datetime(6)"`
	LastBlockIdHash    string                      `gorm:"column:last_block_id_hash;not null;type:varchar(100)"`
	LastCommitHash     string                      `gorm:"column:last_commit_hash;not null;type:varchar(100)"`
	DataHash           string                      `gorm:"column:data_hash;not null;type:varchar(100)"`
	ValidatorsHash     string                      `gorm:"column:validators_hash;not null;type:varchar(100)"`
	NextValidatorsHash string                      `gorm:"column:next_validators_hash;not null;type:varchar(100)"`
	ConsensusHash      string                      `gorm:"column:consensus_hash;not null;type:varchar(100)"`
	AppHash            string                      `gorm:"column:app_hash;not null;type:varchar(100)"`
	LastResultsHash    string                      `gorm:"column:last_results_hash;not null;type:varchar(100)"`
	EvidenceHash       string                      `gorm:"column:evidence_hash;not null;type:varchar(100)"`
	ProposerAddress    string                      `gorm:"column:proposer_address;not null;type:varchar(100)"`
	Round              int32                       `gorm:"column:round;not null;type:int"`
	CommitBlockIdHash  string                      `gorm:"column:commit_block_id_hash;not null;type:varchar(100)"`
	Signatures         []TendermintCommitSignature `gorm:"foreignKey:TendermintCommitCreatedAt,EventUUID;references:CreatedAt,EventUUID"`
}

func (*TendermintCommit) TableName() string {
	return "tendermint_commit"
}

func (s *TendermintCommit) BeforeCreate(tx *gorm.DB) (err error) {
	for _, signature := range s.Signatures {
		err = tx.Create(&signature).Error
	}
	err = tx.Create(&s.Event).Error
	return
}

type TendermintCommitSignature struct {
	ValidatorAddress          string           `gorm:"primaryKey;column:validator_address;not null;type:varchar(100)"`
	TendermintCommit          TendermintCommit `gorm:"foreignKey:TendermintCommitCreatedAt,EventUUID;references:CreatedAt,EventUUID"`
	TendermintCommitCreatedAt time.Time        `gorm:"primaryKey;column:tendermint_commit_created_at;not null;type:datetime(6)"`
	Event                     Event            `gorm:"foreignKey:EventUUID;references:EventUUID"`
	EventUUID                 string           `gorm:"primaryKey;column:event_uuid;not null;type:CHAR(36)"`
	Timestamp                 time.Time        `gorm:"column:timestamp;not null;type:datetime(6)"`
	Signature                 string           `gorm:"column:signature;not null;type:varchar(200)"`
	BlockIdFlag               int              `gorm:"column:block_id_flag;not null;type:int"`
}

func (*TendermintCommitSignature) TableName() string {
	return "tendermint_commit_signature"
}

func (r *EventRepository) FetchHighestHeight(agentName, commitId string) (uint64, error) {
	var (
		maxHeight uint64
	)
	err := r.DB.Raw(`select /*+ USE INDEX (tm INDEX_event_uuid_height) */ max(tm.height)
from tendermint_commit as tm, event as e
where tm.event_uuid = e.event_uuid
and e.agent_name = ?
and e.commit_id = ?
order by tm.height desc
limit 1;`, agentName, commitId).Scan(&maxHeight).Error

	if err != nil {
		return 0, errors.New(fmt.Sprintf("failed to get maximum height: %v", err))
	}

	return maxHeight, nil
}

type ValidatorAddressesWithAgents struct {
	AgentName        string    `gorm:"column:agent_name"`
	EventUUID        string    `gorm:"column:event_uuid"`
	CreatedAt        time.Time `gorm:"column:created_at;not null;type:datetime(6)"`
	Height           uint64    `gorm:"column:height"`
	ValidatorAddress string    `gorm:"column:validator_address;null"`
}

func (r *EventRepository) FindValidatorAddressesWithAgents(validatorAddress string, limit int, agentName string) ([]ValidatorAddressesWithAgents, error) {

	var result []ValidatorAddressesWithAgents
	err := r.DB.Raw(`SELECT /*+ JOIN_ORDER(tc, e, tcs) */
    e.agent_name,
    tc.event_uuid,
    tc.created_at,
    tc.height,
    tcs.validator_address
FROM
    (select /*+ USE_INDEX(INDEX_agent_name_service_name_commit_id_event_uuid) */ agent_name, event_uuid
     from event
     WHERE commit_id = ?
       AND agent_name = ?
       AND service_name = 'tendermint'
       AND created_at >= date_sub(now(), INTERVAL 30 MINUTE)) as e
        JOIN (
            select created_at, event_uuid, height
            from tendermint_commit
            where created_at >= date_sub(now(), INTERVAL 30 MINUTE)
            order by created_at desc) as tc
            ON e.event_uuid = tc.event_uuid
        LEFT JOIN tendermint_commit_signature tcs
            ON tc.event_uuid = tcs.event_uuid
                   AND tc.created_at = tcs.tendermint_commit_created_at
                   AND tcs.validator_address = ?
ORDER BY
    tc.height DESC
LIMIT ?;
`, r.CommitId, agentName, validatorAddress, limit).Scan(&result).Error

	if err != nil {
		return nil, err
	}

	return result, nil

}

type TendermintNodeInfo struct {
	TendermintNodeInfoUUID string `gorm:"primaryKey;column:tendermint_node_info_uuid;not null;type:CHAR(36)"`

	NodeId     string `gorm:"column:node_id;not null;type:varchar(100)"`
	ListenAddr string `gorm:"column:listen_addr;not null;type:varchar(255)"`
	ChainId    string `gorm:"column:chain_id;not null;type:varchar(20)"`
	Moniker    string `gorm:"column:moniker;not null;type:varchar(50)"`

	TendermintPeerInfos []TendermintPeerInfo `gorm:"foreignKey:TendermintNodeInfoUUID;references:TendermintNodeInfoUUID"`
	TendermintNodeInfos []TendermintNodeInfo `gorm:"foreignKey:TendermintNodeInfoUUID;references:TendermintNodeInfoUUID"`
}

func (*TendermintNodeInfo) TableName() string {
	return "tendermint_node_info"
}

type TendermintStatus struct {
	CreatedAt              time.Time          `gorm:"primaryKey;column:created_at;not null;type:datetime(6);autoCreateTime:false"`
	Event                  Event              `gorm:"foreignKey:EventUUID;references:EventUUID"`
	EventUUID              string             `gorm:"primaryKey;column:event_uuid;not null;type:CHAR(36)"`
	TendermintNodeInfo     TendermintNodeInfo `gorm:"foreignKey:TendermintNodeInfoUUID;references:TendermintNodeInfoUUID"`
	TendermintNodeInfoUUID string             `gorm:"column:tendermint_node_info_uuid;not null;type:CHAR(36)"`
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

func (*TendermintStatus) TableName() string {
	return "tendermint_status"
}

func (s *TendermintStatus) BeforeCreate(tx *gorm.DB) (err error) {
	err = tx.Create(&s.TendermintNodeInfo).Error
	err = tx.Create(&s.Event).Error
	return
}

type TSEvent struct {
	AgentName         string    `gorm:"column:agent_name"`
	EventUUID         string    `gorm:"column:event_uuid"`
	CreatedAt         time.Time `gorm:"column:created_at;not null;type:datetime(6)"`
	LatestBlockHeight uint64    `gorm:"column:latest_block_height"`
	LatestBlockTime   time.Time `gorm:"column:latest_block_time;not null;type:datetime(6)"`
	CatchingUp        bool      `gorm:"column:catching_up;null"`
}

func (r *EventRepository) FindFirstTSEventAfterStartTimeGroupByAgentName(startTime time.Time, agentName, serviceName string) (*TSEvent, error) {
	var result *TSEvent

	err := r.DB.Raw(`SELECT /*+ JOIN_ORDER(e, ts) */
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
ORDER BY e.agent_name,ts.created_at DESC
LIMIT 1
`, startTime, serviceName, agentName, r.CommitId).Scan(&result).Error
	if err != nil {
		return nil, err
	}

	return result, nil
}

type TendermintNetInfo struct {
	CreatedAt           time.Time            `gorm:"primaryKey;column:created_at;not null;type:datetime(6);autoCreateTime:false"`
	Event               Event                `gorm:"foreignKey:EventUUID;references:EventUUID"`
	EventUUID           string               `gorm:"primaryKey;column:event_uuid;not null;type:CHAR(36)"`
	NPeers              int                  `gorm:"column:n_peers;not null;type:int"`
	Listening           bool                 `gorm:"column:listening;not null;type:bool"`
	TendermintPeerInfos []TendermintPeerInfo `gorm:"foreignKey:TendermintNetInfoCreatedAt;references:CreatedAt"`
}

func (*TendermintNetInfo) TableName() string {
	return "tendermint_net_info"
}

func (s *TendermintNetInfo) BeforeCreate(tx *gorm.DB) (err error) {
	for _, p := range s.TendermintPeerInfos {
		err = tx.Create(&p.TendermintNodeInfo).Error
	}
	err = tx.Create(&s.Event).Error
	return
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

func (*TendermintPeerInfo) TableName() string {
	return "tendermint_peer_info"
}

type AgentPeerInfo struct {
	AgentName         string    `gorm:"column:agent_name"`
	EventUUID         string    `gorm:"column:event_uuid"`
	CreatedAt         time.Time `gorm:"column:created_at;not null;type:datetime(6)"`
	NPeers            int       `gorm:"column:n_peers"`
	PeerInfoUUIDCount int       `gorm:"column:tpi_count"`
}

func (r *EventRepository) FindLatestAgentPeerInfosByAgentNameAndStartTime(agentName, eventType, serviceName string, startTime time.Time) ([]AgentPeerInfo, error) {
	var result []AgentPeerInfo

	err := r.DB.Raw(`SELECT /*+ JOIN_ORDER(max_ein, tni, tpi)*/
    max_ein.agent_name as agent_name,
    max_ein.event_uuid as event_uuid,
    tni.created_at as created_at,
    tni.n_peers as n_peers,
    COUNT(tpi.tendermint_peer_info_uuid) AS tpi_count
FROM (
        SELECT
            agent_name,
            event_uuid,
            created_at,
            event_type
        FROM
            event
        WHERE agent_name = ?
          AND service_name = ?
          and commit_id = ?
          AND event_type = ?
          AND created_at >= ?
        order by agent_name, created_at desc
        limit 50
    ) max_ein, tendermint_net_info tni, tendermint_peer_info tpi
WHERE tni.event_uuid = max_ein.event_uuid
  AND tni.created_at = max_ein.created_at
  AND tni.event_uuid = tpi.event_uuid
  AND tni.created_at = tpi.created_at
GROUP BY
    max_ein.agent_name, max_ein.event_uuid, tni.created_at, tni.n_peers
`, agentName, serviceName, r.CommitId, eventType, startTime).Scan(&result).Error

	if err != nil {
		return nil, err
	}

	return result, nil
}
