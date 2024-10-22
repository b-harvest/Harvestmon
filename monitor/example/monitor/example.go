package monitor

import (
	"errors"
	"fmt"
	_const "github.com/b-harvest/Harvestmon/const"
	"github.com/b-harvest/Harvestmon/log"
	"github.com/b-harvest/Harvestmon/monitor/example/types"
	"github.com/b-harvest/Harvestmon/repository"
	"github.com/b-harvest/Harvestmon/util"
	"github.com/google/uuid"
	"strconv"
	"time"
)

func ExampleMonitor(c *types.MonitorConfig, client *types.MonitorClient) {
	_, _, fn := util.TraceFirst()
	log.Debug("Starting monitor: " + fn)

	// define repository to communicate with Database
	netInfoMonitorRepository := repository.NetInfoRepository{BaseRepository: repository.BaseRepository{DB: *client.GetDatabase(c.DbBatchSize)}}

	// API request
	netInfo, err := client.GetNetInfo()
	if err != nil {
		log.Error(err)
		return
	}
	if netInfo == nil {
		log.Error(errors.New("net_info monitor response <nil>"))
		return
	}

	// Store responses to database
	eventUUID, err := uuid.NewUUID()
	if err != nil {
		log.Error(err)
	}

	createdAt := time.Now().UTC()

	var tendermintPeerInfos []repository.TendermintPeerInfo
	for _, peer := range netInfo.Result.Peers {
		tendermintPeerUUID, err := uuid.NewUUID()
		if err != nil {
			log.Error(err)
		}
		tendermintNodeUUID, err := uuid.NewUUID()
		if err != nil {
			log.Error(err)
		}

		tendermintPeerInfos = append(tendermintPeerInfos,
			repository.TendermintPeerInfo{
				TendermintPeerInfoUUID:     tendermintPeerUUID.String(),
				TendermintNetInfoCreatedAt: createdAt,
				EventUUID:                  eventUUID.String(),
				IsOutbound:                 peer.IsOutbound,
				TendermintNodeInfoUUID:     tendermintNodeUUID.String(),
				TendermintNodeInfo: repository.TendermintNodeInfo{
					TendermintNodeInfoUUID: tendermintNodeUUID.String(),
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
		log.Error(err)
	}

	err = netInfoMonitorRepository.Save(
		repository.TendermintNetInfo{
			CreatedAt: createdAt,
			EventUUID: eventUUID.String(),
			Event: repository.Event{
				EventUUID:   eventUUID.String(),
				AgentName:   c.Agent.AgentName,
				ServiceName: _const.HARVESTMON_TENDERMINT_SERVICE_NAME,
				CommitID:    c.Agent.CommitId,
				EventType:   _const.TM_NET_INFO_EVENT_TYPE,
				CreatedAt:   createdAt,
			},
			TendermintPeerInfos: tendermintPeerInfos,
			NPeers:              nPeers,
			Listening:           netInfo.Result.Listening,
		})
	if err != nil {
		log.Warn(err.Error())
	}

	// Completed to store example
	log.Info(fmt.Sprintf("[example] peer_count: %d", nPeers))

	log.Debug("Complete monitor: " + fn)
}
