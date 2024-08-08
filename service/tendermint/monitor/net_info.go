package monitor

import (
	"fmt"
	"github.com/b-harvest/Harvestmon/log"
	"github.com/b-harvest/Harvestmon/repository"
	"github.com/b-harvest/Harvestmon/util"
	"github.com/google/uuid"
	"strconv"
	"tendermint-mon/types"
	"time"
)

func NetInfoMonitor(c *types.MonitorConfig, client *types.MonitorClient) {
	_, _, fn := util.Trace()
	log.Debug("Starting monitor: " + fn)

	netInfoMonitorRepository := repository.NetInfoRepository{EventRepository: repository.EventRepository{DB: *client.GetDatabase()}}

	netInfo, err := client.GetNetInfo()
	if err != nil {
		log.Error(err)
	}

	eventUUID, err := uuid.NewUUID()
	if err != nil {
		log.Error(err)
	}

	createdAt := time.Now()

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
				ServiceName: types.HARVEST_SERVICE_NAME,
				CommitID:    c.Agent.CommitId,
				EventType:   types.TM_NET_INFO_EVENT_TYPE,
				CreatedAt:   createdAt,
			},
			TendermintPeerInfos: tendermintPeerInfos,
			NPeers:              nPeers,
			Listening:           netInfo.Result.Listening,
		})
	if err != nil {
		log.Warn(err.Error())
	}
	log.Info(fmt.Sprintf("[net_info] peer_count: %d", nPeers))

	log.Debug("Complete monitor: " + fn)
}
