package checker

import (
	"errors"
	"github.com/b-harvest/Harvestmon/log"
	"github.com/b-harvest/Harvestmon/repository"
	"tendermint-checker/alarmer"
	"tendermint-checker/types"
	"time"
)

func NetInfoChecker(c *types.CheckerConfig, client *types.CheckerClient) {
	netInfoRepository := repository.NetInfoRepository{EventRepository: repository.EventRepository{DB: *client.GetDatabase(), CommitId: c.CommitId}}

	agentPeerInfos, err := netInfoRepository.FindLatestAgentPeerInfos()
	if err != nil {
		log.Error(errors.New(netInfoFormatf(err.Error())))
	}

	for _, agentPeerInfo := range agentPeerInfos {
		if agentPeerInfo.CreatedAt.Add(5 * time.Minute).Before(time.Now()) {
			log.Warn(netInfoFormatf("Agent(%s)'s latest peer info is too old: %v (%s ago)", agentPeerInfo.AgentName, agentPeerInfo.CreatedAt, time.Now().Sub(agentPeerInfo.CreatedAt)))
		}
		if agentPeerInfo.NPeers != agentPeerInfo.PeerInfoUUIDCount {
			log.Warn(netInfoFormatf("It is different with NPeers and length of PeerInfos. You should check it. EventUUID: %s", agentPeerInfo.EventUUID))
		}
		if agentPeerInfo.NPeers < c.PeerCheck.LowPeerCount {
			if alertLevel, exists := client.AlertLevelList[LOW_PEER_TM_ALARM_TYPE]; exists {
				for _, a := range alertLevel.AlarmerList {

					// Pass to alarmer
					alarmer.RunAlarm(alarmer.NewAlert(a))
				}

			} else {
				log.Error(errors.New(netInfoFormatf("Cannot find alarm level: %s", LOW_PEER_TM_ALARM_TYPE)))
			}
		}

		log.Debug(netInfoFormatf("Complete to check Agent: (%s).", agentPeerInfo.AgentName))
	}

}
