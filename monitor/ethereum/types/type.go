package types

import (
	"time"
)

type Monitor interface {
	Run(c *MonitorConfig, rpcClient *MonitorClient)
}

type Func struct {
	MonitorFunc `yaml:"name"`
	Interval    *time.Duration `yaml:"interval"`
}

type MonitorFunc func(c *MonitorConfig, rpcClient *MonitorClient)

func (f Func) Run(c *MonitorConfig, rpcClient *MonitorClient) {
	f.MonitorFunc(c, rpcClient)
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

type CometBFTNetInfoResult struct {
	Result ResultNetInfo `json:"result"`
	// By default, most tendermint chain returns id as "-1". but some other ones are not.
	// if id's type is set as int64(or other integer/string types), it'll throw unmarshaling error.
	ID      any    `json:"id"`
	Jsonrpc string `json:"jsonrpc"`
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
