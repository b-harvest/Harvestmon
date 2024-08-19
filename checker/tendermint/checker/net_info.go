package checker

import (
	"errors"
	"fmt"
	"github.com/b-harvest/Harvestmon/checker/tendermint/alarmer"
	"github.com/b-harvest/Harvestmon/checker/tendermint/types"
	"github.com/b-harvest/Harvestmon/const"
	"github.com/b-harvest/Harvestmon/log"
	"github.com/b-harvest/Harvestmon/repository"
	"github.com/b-harvest/Harvestmon/util"
	"time"
)

func NetInfoChecker(c *types.CheckerConfig, client *types.CheckerClient) {
	_, _, fn := util.TraceFirst()
	log.Debug(netInfoFormatf("Starting: " + fn))

	netInfoRepository := repository.NetInfoRepository{BaseRepository: repository.BaseRepository{DB: *client.GetDatabase(), CommitId: c.CommitId}}

	for agentName, agentChecker := range c.AgentCheckers {
		agentPeerInfos, err := netInfoRepository.FindLatestAgentPeerInfosByAgentName(string(agentName), _const.TM_NET_INFO_EVENT_TYPE, _const.HARVESTMON_TENDERMINT_SERVICE_NAME)
		if err != nil {
			log.Error(errors.New(netInfoFormatf(err.Error())))
		}

		for _, agentPeerInfo := range agentPeerInfos {
			if agentPeerInfo.CreatedAt.Add(5 * time.Minute).Before(time.Now().UTC()) {
				log.Warn(netInfoFormatf("Agent(%s)'s latest peer info is too old: %v (%s ago)", agentPeerInfo.AgentName, agentPeerInfo.CreatedAt, time.Now().Sub(agentPeerInfo.CreatedAt)))
			}
			if agentPeerInfo.NPeers != agentPeerInfo.PeerInfoUUIDCount {
				log.Warn(netInfoFormatf("It is different with NPeers and length of PeerInfos. You should check it. EventUUID: %s", agentPeerInfo.EventUUID))
			}
			if agentPeerInfo.NPeers < agentChecker.PeerCheck.LowPeerCount {
				var errorMsg = fmt.Sprintf("\nCurrent Peer Count: %d\nThresholdPeer: %d", agentPeerInfo.NPeers, agentChecker.PeerCheck.LowPeerCount)

				var (
					alertLevel = client.GetAlertLevelList(agentName, string(HEARTBEAT_TM_ALARM_TYPE))
					sent       bool
				)
				// Exceeded max missing count.

				for _, a := range client.GetAlarmerList(agentName, alertLevel.AlertLevel) {
					sent = true

					// Pass to alarmer
					err = alarmer.RunAlarm(c, *client, types.NewAlert(a, alertLevel, agentName, errorMsg))
					if err != nil {
						log.Error(errors.New(netInfoFormatf("error occurred while sending alarm: %s, %v", LOW_PEER_TM_ALARM_TYPE, err)))
					}
				}
				if !sent {
					log.Error(errors.New(netInfoFormatf("Didn't send any alert cause of no alarmer specified for the level: %s, %s", LOW_PEER_TM_ALARM_TYPE, alertLevel.AlertLevel)))
				}

			}

			log.Debug(netInfoFormatf("Complete to check Agent: (%s).", agentPeerInfo.AgentName))
		}
	}

}
