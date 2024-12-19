package main

import (
	"context"
	"encoding/json"
	"fmt"
	_const "github.com/b-harvest/Harvestmon/const"
	"github.com/b-harvest/Harvestmon/repository"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	TM_COMMIT_MAX_WINDOW_SIZE = 2000
	TM_COMMIT_RESET_SIZE      = TM_COMMIT_MAX_WINDOW_SIZE * 2
)

type TendermintMonitorConfig struct {
	logger *log.Entry

	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`

	agentName string
	commitId  string

	monitorTarget MonitorTarget
	Collectors    []Collector `mapstructure:"collectors"`

	client *http.Client

	batchSize int

	height          uint64
	lastStoreCommit uint64
}

func (c *TendermintMonitorConfig) Validate() error {
	if c.logger == nil {
		return errors.New("logger is required")
	}

	if c.Host == "" {
		return errors.New("host is required")
	}

	if c.Port == 0 {
		return errors.New("port is required")
	}

	if c.agentName == "" {
		return errors.New("agent name is required")
	}

	if c.commitId == "" {
		return errors.New("commitId is required")
	}

	if len(c.Collectors) == 0 {
		return errors.New("collectors is required")
	}

	if c.batchSize == 0 {
		return errors.New("batchSize is required")
	}

	if c.client == nil {
		return errors.New("client is required")
	}

	for _, collector := range c.Collectors {
		if err := collector.Validate(); err != nil {
			return errors.Wrapf(err, "collector '%v' is invalid", collector.Name)
		}
	}

	return nil
}

func (c *TendermintMonitorConfig) initialize(
	agentName, commitId string,
	batchSize int,
	logger *log.Entry,
	client *http.Client,
	collectors []Collector) {

	c.agentName = agentName
	c.commitId = commitId
	c.batchSize = batchSize
	c.logger = logger
	c.client = client
	c.Collectors = collectors
}

func (c *TendermintMonitorConfig) getMonitorName() string {
	return _const.HARVESTMON_TENDERMINT_SERVICE_NAME
}

func (c *TendermintMonitorConfig) getLogger() *log.Entry {
	return c.logger
}

func (c *TendermintMonitorConfig) getCollectors() []Collector {
	return c.Collectors
}

func (c *TendermintMonitorConfig) load(r *repository.BaseRepository) error {
	repo := repository.MetaMonitorRepository{BaseRepository: *r}
	height, err := repo.FetchHighestHeight(c.agentName)
	c.height = height
	if height == 0 || err != nil {
		c.lastStoreCommit = 0
	} else {
		c.lastStoreCommit = height
	}
	return nil
}

func (c *TendermintMonitorConfig) exit(repo *repository.BaseRepository) error {
	m := repository.MetaMonitorRepository{BaseRepository: *repo}
	return m.Save(repository.MetaMonitor{
		AgentName: c.agentName,
		Height:    int64(c.lastStoreCommit),
	})
}

var tmStatusCollector = func(mc MonitorConfig) func(sq chan StoreEntity, l *log.Entry) {
	c, ok := mc.(*TendermintMonitorConfig)
	if !ok {
		return func(sq chan StoreEntity, l *log.Entry) {
			l.Warningf("invalid config type")
		}
	}

	return func(sq chan StoreEntity, l *log.Entry) {

		endpoint := getEndpoint(c.Host, c.Port)
		endpoint += "/status"

		ctx, cancel := context.WithTimeout(context.Background(), c.client.Timeout)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			l.Warningf(errors.Wrapf(err, "failed to build request '%s'", endpoint).Error())
			return
		}

		var (
			body         []byte
			statusResult CometBFTStatusResult
		)
		body, err = request(c.client, req, 3)
		if err != nil {
			l.Warningf(errors.Wrapf(err, "failed to request to endpoint").Error())
			return
		}

		err = json.Unmarshal(body, &statusResult)
		if err != nil {
			l.Warningf(err.Error())
			return
		}

		tmStatus, err := newTendermintStatus(c.agentName, c.commitId, statusResult)
		if err != nil {
			l.Error(err.Error())
			return
		}
		sq <- StoreEntity{
			tmStatus: tmStatus,
		}

		c.logger.Debugf("got tendermint status height: %v", tmStatus.LatestBlockHeight)
		c.height = tmStatus.LatestBlockHeight

		return
	}
}

var tmNetInfoCollector = func(mc MonitorConfig) func(sq chan StoreEntity, l *log.Entry) {
	c, ok := mc.(*TendermintMonitorConfig)
	if !ok {
		return func(sq chan StoreEntity, l *log.Entry) {
			l.Warningf("invalid config type")
			return
		}
	}

	return func(sq chan StoreEntity, l *log.Entry) {
		endpoint := getEndpoint(c.Host, c.Port)
		endpoint += "/net_info"

		ctx, cancel := context.WithTimeout(context.Background(), c.client.Timeout)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			l.Warningf(errors.Wrapf(err, "failed to build reqest '%s'", endpoint).Error())
			return
		}

		var (
			body          []byte
			netInfoResult CometBFTNetInfoResult
		)
		body, err = request(c.client, req, 3)
		if err != nil {
			l.Warningf(errors.Wrapf(err, "failed to request '%s'", endpoint).Error())
			return
		}

		err = json.Unmarshal(body, &netInfoResult)
		if err != nil {
			l.Warningf(errors.Wrapf(err, "failed to unmarshal response '%s'", endpoint).Error())
			return
		}

		tmNetInfo, err := newTendermintNetInfo(c.agentName, c.commitId, netInfoResult)
		if err != nil {
			l.Error(errors.Wrapf(err, "failed to parse net_info response '%s'", endpoint).Error())
			return
		}

		c.logger.Debugf("got netInfo n_peers: %v", tmNetInfo.NPeers)
		sq <- StoreEntity{
			tmNetInfo: tmNetInfo,
		}

		return
	}
}

var tmCommitCollector = func(mc MonitorConfig) func(sq chan StoreEntity, l *log.Entry) {
	c, ok := mc.(*TendermintMonitorConfig)
	if !ok {
		return func(sq chan StoreEntity, l *log.Entry) {
			l.Warningf("invalid config type")
		}
	}

	return func(sq chan StoreEntity, l *log.Entry) {

		untilHeight := c.height
		startHeight := c.lastStoreCommit
		if startHeight == 0 && untilHeight != 0 {
			startHeight = untilHeight - 1
		}

		if untilHeight <= startHeight {
			l.Warningf("height should be greater than start height. until: %v, start: %v", untilHeight, startHeight)
			return
		} else if untilHeight-startHeight > TM_COMMIT_MAX_WINDOW_SIZE { // filter
			if untilHeight-startHeight > TM_COMMIT_RESET_SIZE {
				startHeight = untilHeight - TM_COMMIT_MAX_WINDOW_SIZE
			} else {
				untilHeight = startHeight + TM_COMMIT_MAX_WINDOW_SIZE
			}

		}

		// findWindowSize function finds proper window size to regulate concurrency.
		// it'll be less than 40 (it's fixed value)
		var findWindowSize func(w uint64) uint64
		findWindowSize = func(w uint64) uint64 {
			if w > 40 {
				w /= 2
				findWindowSize(w)
			}
			return w
		}

		windowSize := findWindowSize(untilHeight - startHeight)

		l.Debugf("start commit collector, starting: %v, until: %v, windowSize: %v", startHeight, untilHeight, windowSize)

		var highestHeight uint64

		// to prevent collecting uncompleted block,
		// it doesn't fetch latest height block commit.
		for i := startHeight; i < untilHeight; i += uint64(c.batchSize) {
			var (
				wg         sync.WaitGroup
				recordChan = make(chan CometBFTCommitResult, windowSize)
			)

			semaphore := make(chan struct{}, windowSize)

			// Adjust the upper bound to avoid fetching heights beyond latestHeight
			batchEnd := i + windowSize
			if batchEnd > untilHeight {
				batchEnd = untilHeight
			}

			for j := i; j < batchEnd; j++ {
				wg.Add(1)
				go func(height uint64) {
					defer wg.Done()
					// semaphore to prevent overload
					semaphore <- struct{}{}
					defer func() { <-semaphore }()

					endpoint := getEndpoint(c.Host, c.Port)
					endpoint += "/commit"

					ctx, cancel := context.WithTimeout(context.Background(), c.client.Timeout)
					defer cancel()
					req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s?height=%d", endpoint, height), nil)
					if err != nil {
						l.Warningf(errors.Wrapf(err, "failed to fetch commit info for height %d", height).Error())
						return
					}

					var (
						body         []byte
						commitResult CometBFTCommitResult
					)
					body, err = request(c.client, req, 3)
					if err != nil {
						l.Warningf(errors.Wrapf(err, "failed to build reqest").Error())
						return
					}

					err = json.Unmarshal(body, &commitResult)
					if err != nil {
						l.Warningf(errors.Wrapf(err, "failed to marshal body").Error())
						return
					}
					recordChan <- commitResult
				}(j)
			}

			go func() {
				wg.Wait()
				close(recordChan)
			}()

			for record := range recordChan {
				tc, err := newTendermintCommit(
					c.agentName, c.commitId, record)

				height, _ := strconv.ParseUint(record.Result.Height, 10, 64)
				if highestHeight < height {
					highestHeight = height
				}
				l.Debugf("commit collector, height: %v", height)
				if err != nil {
					l.Warningf(errors.Wrapf(err, "failed to parse commit info for height %d", height).Error())
					continue
				}

				sq <- StoreEntity{
					tmCommit: tc,
				}
			}

		}

		c.lastStoreCommit = highestHeight + 1

		return
	}
}

func newTendermintStatus(agentName, commitId string, status CometBFTStatusResult) (*repository.TendermintStatus, error) {
	eventUUID, err := uuid.NewUUID()
	if err != nil {
		return nil, err
	}
	nodeInfoUUID, err := uuid.NewUUID()
	if err != nil {
		return nil, err
	}

	createdAt := time.Now().UTC()

	latestBlockHeight, err := strconv.ParseUint(status.Result.SyncInfo.LatestBlockHeight, 0, 64)
	if err != nil {
		return nil, errors.New("LatestBlockHeight Parsing error: " + status.Result.SyncInfo.LatestBlockHeight + ". err: " + err.Error())
	}
	earliestBlockHeight, err := strconv.ParseUint(status.Result.SyncInfo.EarliestBlockHeight, 0, 64)
	if err != nil {
		return nil, errors.New("EarliestBLockHeight Parsing error at " + status.Result.SyncInfo.EarliestBlockHeight + ". it automatically set as 0. err: " + err.Error())
	}

	return &repository.TendermintStatus{
		CreatedAt: createdAt,
		EventUUID: eventUUID.String(),
		Event: repository.Event{
			EventUUID:   eventUUID.String(),
			AgentName:   agentName,
			ServiceName: _const.HARVESTMON_TENDERMINT_SERVICE_NAME,
			CommitID:    commitId,
			EventType:   _const.TM_STATUS_EVENT_TYPE,
			CreatedAt:   createdAt,
		},
		TendermintNodeInfoUUID: nodeInfoUUID.String(),
		TendermintNodeInfo: repository.TendermintNodeInfo{
			TendermintNodeInfoUUID: nodeInfoUUID.String(),
			NodeId:                 string(status.Result.NodeInfo.DefaultNodeID),
			ListenAddr:             status.Result.NodeInfo.ListenAddr,
			ChainId:                status.Result.NodeInfo.Network,
			Moniker:                status.Result.NodeInfo.Moniker,
		},
		LatestBlockHash:     string(status.Result.SyncInfo.LatestBlockHash),
		LatestAppHash:       string(status.Result.SyncInfo.LatestAppHash),
		LatestBlockHeight:   latestBlockHeight,
		LatestBlockTime:     status.Result.SyncInfo.LatestBlockTime,
		EarliestBlockHash:   string(status.Result.SyncInfo.EarliestBlockHash),
		EarliestAppHash:     string(status.Result.SyncInfo.EarliestAppHash),
		EarliestBlockHeight: earliestBlockHeight,
		EarliestBlockTime:   status.Result.SyncInfo.EarliestBlockTime,
		CatchingUp:          status.Result.SyncInfo.CatchingUp,
	}, nil
}

func newTendermintNetInfo(agentName, commitId string, netInfo CometBFTNetInfoResult) (*repository.TendermintNetInfo, error) {
	eventUUID, err := uuid.NewUUID()
	if err != nil {
		return nil, err
	}

	createdAt := time.Now().UTC()

	var (
		tendermintPeerInfos []repository.TendermintPeerInfo
		pid                 uuid.UUID
		nid                 uuid.UUID
	)
	for _, peer := range netInfo.Result.Peers {
		pid, err = uuid.NewUUID()
		if err != nil {
			return nil, err
		}
		nid, err = uuid.NewUUID()
		if err != nil {
			return nil, err
		}

		tendermintPeerInfos = append(tendermintPeerInfos,
			repository.TendermintPeerInfo{
				TendermintPeerInfoUUID:     pid.String(),
				TendermintNetInfoCreatedAt: createdAt,
				EventUUID:                  eventUUID.String(),
				IsOutbound:                 peer.IsOutbound,
				TendermintNodeInfoUUID:     nid.String(),
				TendermintNodeInfo: repository.TendermintNodeInfo{
					TendermintNodeInfoUUID: nid.String(),
					NodeId:                 string(peer.NodeInfo.DefaultNodeID),
					ListenAddr:             peer.NodeInfo.ListenAddr,
					ChainId:                peer.NodeInfo.Network,
					Moniker:                peer.NodeInfo.Moniker,
				},
				RemoteIP: peer.RemoteIP,
			})

	}

	nPeers, err := strconv.Atoi(netInfo.Result.NPeers)
	if err != nil {
		return nil, err
	}

	return &repository.TendermintNetInfo{
		CreatedAt: createdAt,
		EventUUID: eventUUID.String(),
		Event: repository.Event{
			EventUUID:   eventUUID.String(),
			AgentName:   agentName,
			ServiceName: _const.HARVESTMON_TENDERMINT_SERVICE_NAME,
			CommitID:    commitId,
			EventType:   _const.TM_NET_INFO_EVENT_TYPE,
			CreatedAt:   createdAt,
		},
		TendermintPeerInfos: tendermintPeerInfos,
		NPeers:              nPeers,
		Listening:           netInfo.Result.Listening,
	}, nil
}

func newTendermintCommit(agentName, commitId string, commit CometBFTCommitResult) (*repository.TendermintCommit, error) {
	eventUUID, err := uuid.NewUUID()
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Error generating UUID: %v", err))
	}

	createdAt := time.Now().UTC()

	var signatures []repository.TendermintCommitSignature

	var commitSigs = commit.Result.SignedHeader.Commit.Signatures
	if len(commitSigs) == 0 && len(commit.Result.SignedHeader.Commit.Precommits) > 0 {
		commitSigs = commit.Result.SignedHeader.Commit.Precommits
	}

	for _, signature := range commitSigs {
		if signature.ValidatorAddress == "" {
			continue
		}
		signatures = append(signatures, repository.TendermintCommitSignature{
			ValidatorAddress:          signature.ValidatorAddress,
			TendermintCommitCreatedAt: createdAt,
			EventUUID:                 eventUUID.String(),
			Timestamp:                 signature.Timestamp,
			Signature:                 signature.Signature,
			BlockIdFlag:               signature.BlockIDFlag,
		})
	}

	return &repository.TendermintCommit{
		CreatedAt: createdAt,
		EventUUID: eventUUID.String(),
		Event: repository.Event{
			EventUUID:   eventUUID.String(),
			AgentName:   agentName,
			ServiceName: _const.HARVESTMON_TENDERMINT_SERVICE_NAME,
			CommitID:    commitId,
			EventType:   _const.TM_COMMIT_EVENT_TYPE,
			CreatedAt:   createdAt,
		},
		ChainID:            commit.Result.ChainID,
		Height:             commit.Result.Height,
		Time:               commit.Result.Time,
		LastBlockIdHash:    commit.Result.LastBlockID.Hash,
		LastCommitHash:     commit.Result.LastCommitHash,
		DataHash:           commit.Result.DataHash,
		ValidatorsHash:     commit.Result.ValidatorsHash,
		NextValidatorsHash: commit.Result.NextValidatorsHash,
		ConsensusHash:      commit.Result.ConsensusHash,
		AppHash:            commit.Result.AppHash,
		LastResultsHash:    commit.Result.LastResultsHash,
		EvidenceHash:       commit.Result.EvidenceHash,
		ProposerAddress:    commit.Result.ProposerAddress,
		Round:              commit.Result.Commit.Round,
		CommitBlockIdHash:  commit.Result.Commit.BlockID.Hash,
		Signatures:         signatures,
	}, nil

}

type CometBFTStatusResult struct {
	Result ResultStatus `json:"result"`
	// By default, most tendermint chain returns id as "-1". but some other ones are not.
	// if id's type is set as int64(or other integer/string types), it'll throw unmarshaling error.
	ID      any    `json:"id"`
	Jsonrpc string `json:"jsonrpc"`
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

type CometBFTCommitResult struct {
	Result ResultCommit `json:"result"`
	// By default, most tendermint chain returns id as "-1". but some other ones are not.
	// if id's type is set as int64(or other integer/string types), it'll throw unmarshaling error.
	ID      any    `json:"id"`
	Jsonrpc string `json:"jsonrpc"`
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
	Total any    `json:"total"`
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
	Precommits []CommitSig `json:"precommits"`

	hash string
}

type CommitSig struct {
	BlockIDFlag      int       `json:"block_id_flag"`
	ValidatorAddress string    `json:"validator_address"`
	Timestamp        time.Time `json:"timestamp"`
	Signature        string    `json:"signature"`
}
