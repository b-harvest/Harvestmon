package types

import (
	"time"
)

type Monitor interface {
	Run(c *MonitorConfig, rpcClient *MonitorClient)
}

type Func func(c *MonitorConfig, rpcClient *MonitorClient)

func (f Func) Run(c *MonitorConfig, rpcClient *MonitorClient) {
	f(c, rpcClient)
}

const (
	TM_EVENT_TYPE          = "tm:event"
	TM_STATUS_EVENT_TYPE   = TM_EVENT_TYPE + ":status"
	TM_NET_INFO_EVENT_TYPE = TM_EVENT_TYPE + ":net_info"
	TM_COMMIT_EVENT_TYPE   = TM_EVENT_TYPE + ":commit"
)

type CometBFTStatusResult struct {
	Result  ResultStatus `json:"result"`
	ID      int64        `json:"id"`
	Jsonrpc string       `json:"jsonrpc"`
}

// Node Status
type ResultStatus struct {
	NodeInfo      DefaultNodeInfo `json:"node_info"`
	SyncInfo      SyncInfo        `json:"sync_info"`
	ValidatorInfo ValidatorInfo   `json:"validator_info"`
}

type DefaultNodeInfo struct {
	ProtocolVersion ProtocolVersion `json:"protocol_version"`

	// Authenticate
	// TODO: replace with NetAddress
	DefaultNodeID string `json:"id"`          // authenticated identifier
	ListenAddr    string `json:"listen_addr"` // accepting incoming

	// Check compatibility.
	// Channels are HexBytes so easier to read as JSON
	Network  string   `json:"network"`  // network/chain ID
	Version  string   `json:"version"`  // major.minor.revision
	Channels HexBytes `json:"channels"` // channels this node knows about

	// ASCIIText fields
	Moniker string               `json:"moniker"` // arbitrary moniker
	Other   DefaultNodeInfoOther `json:"other"`   // other application specific data
}

// ProtocolVersion contains the protocol versions for the software.
type ProtocolVersion struct {
	P2P   string `json:"p2p"`
	Block string `json:"block"`
	App   string `json:"app"`
}
type ID string

type HexBytes string

type DefaultNodeInfoOther struct {
	TxIndex    string `json:"tx_index"`
	RPCAddress string `json:"rpc_address"`
}

type SyncInfo struct {
	LatestBlockHash   HexBytes  `json:"latest_block_hash"`
	LatestAppHash     HexBytes  `json:"latest_app_hash"`
	LatestBlockHeight string    `json:"latest_block_height"`
	LatestBlockTime   time.Time `json:"latest_block_time"`

	EarliestBlockHash   HexBytes  `json:"earliest_block_hash"`
	EarliestAppHash     HexBytes  `json:"earliest_app_hash"`
	EarliestBlockHeight string    `json:"earliest_block_height"`
	EarliestBlockTime   time.Time `json:"earliest_block_time"`

	CatchingUp bool `json:"catching_up"`
}

type ValidatorInfo struct {
	Address     HexBytes `json:"address"`
	PubKey      any      `json:"pub_key"`
	VotingPower string   `json:"voting_power"`
}

type CometBFTNetInfoResult struct {
	Result  ResultNetInfo `json:"result"`
	ID      int64         `json:"id"`
	Jsonrpc string        `json:"jsonrpc"`
}

type ResultNetInfo struct {
	Listening bool     `json:"listening"`
	Listeners []string `json:"listeners"`
	NPeers    string   `json:"n_peers"`
	Peers     []Peer   `json:"peers"`
}

type Peer struct {
	NodeInfo         DefaultNodeInfo  `json:"node_info"`
	IsOutbound       bool             `json:"is_outbound"`
	ConnectionStatus ConnectionStatus `json:"connection_status"`
	RemoteIP         string           `json:"remote_ip"`
}

type ConnectionStatus struct {
	Duration    string
	SendMonitor Status
	RecvMonitor Status
	Channels    []ChannelStatus
}

type ChannelStatus struct {
	ID                byte
	SendQueueCapacity string
	SendQueueSize     string
	Priority          string
	RecentlySent      string
}

type Percent uint32

type Status struct {
	Start    time.Time // Transfer start time
	Bytes    string    // Total number of bytes transferred
	Samples  string    // Total number of samples taken
	InstRate string    // Instantaneous transfer rate
	CurRate  string    // Current transfer rate (EMA of InstRate)
	AvgRate  string    // Average transfer rate (Bytes / Duration)
	PeakRate string    // Maximum instantaneous transfer rate
	BytesRem string    // Number of bytes remaining in the transfer
	Duration string    // Time period covered by the statistics
	Idle     string    // Time since the last transfer of at least 1 byte
	TimeRem  string    // Estimated time to completion
	Progress Percent   // Overall transfer progress
	Active   bool      // Flag indicating an active transfer
}

type CometBFTCommitResult struct {
	Result  ResultCommit `json:"result"`
	ID      int64        `json:"id"`
	Jsonrpc string       `json:"jsonrpc"`
}

type ResultCommit struct {
	SignedHeader    `json:"signed_header"`
	CanonicalCommit bool `json:"canonical"`
}

type SignedHeader struct {
	*Header `json:"header"`

	Commit *Commit `json:"commit"`
}

type Header struct {
	// basic block info
	Version Consensus `json:"version"`
	ChainID string    `json:"chain_id"`
	Height  string    `json:"height"`
	Time    time.Time `json:"time"`

	// prev block info
	LastBlockID BlockID `json:"last_block_id"`

	// hashes of block data
	LastCommitHash string `json:"last_commit_hash"` // commit from validators from the last block
	DataHash       string `json:"data_hash"`        // transactions

	// hashes from the app output from the prev block
	ValidatorsHash     string `json:"validators_hash"`      // validators for the current block
	NextValidatorsHash string `json:"next_validators_hash"` // validators for the next block
	ConsensusHash      string `json:"consensus_hash"`       // consensus params for current block
	AppHash            string `json:"app_hash"`             // state after txs from the previous block
	// root hash of all results from the txs from the previous block
	// see `deterministicExecTxResult` to understand which parts of a tx is hashed into here
	LastResultsHash string `json:"last_results_hash"`

	// consensus info
	EvidenceHash    string `json:"evidence_hash"`    // evidence included in the block
	ProposerAddress string `json:"proposer_address"` // original proposer of the block
}

type Consensus struct {
	Block string `protobuf:"varint,1,opt,name=block,proto3" json:"block,omitempty"`
	App   string `protobuf:"varint,2,opt,name=app,proto3" json:"app,omitempty"`
}

type BlockID struct {
	Hash          string        `json:"hash"`
	PartSetHeader PartSetHeader `json:"parts"`
}

type PartSetHeader struct {
	Total uint32 `json:"total"`
	Hash  string `json:"hash"`
}

type Commit struct {
	// NOTE: The signatures are in order of address to preserve the bonded
	// ValidatorSet order.
	// Any peer with a block can gossip signatures by index with a peer without
	// recalculating the active ValidatorSet.
	Height     string      `json:"height"`
	Round      int32       `json:"round"`
	BlockID    BlockID     `json:"block_id"`
	Signatures []CommitSig `json:"signatures"`

	hash string
}

type CommitSig struct {
	BlockIDFlag      int       `json:"block_id_flag"`
	ValidatorAddress string    `json:"validator_address"`
	Timestamp        time.Time `json:"timestamp"`
	Signature        string    `json:"signature"`
}
