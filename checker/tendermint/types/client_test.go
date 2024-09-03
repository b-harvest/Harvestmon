package types

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test(t *testing.T) {

	var (
		agentName          AgentName = "test-agent"
		heartbeatAlertName           = AlertName("tendermint:heartbeat")
	)

	client := CheckerClient{
		AgentAlertLevelList: make(map[AgentName]map[AlertName]AlertLevel),
	}

	t.Run("GetAlertLevelTest - happy path single", func(t *testing.T) {
		client.AgentAlertLevelList = map[AgentName]map[AlertName]AlertLevel{
			agentName: {
				heartbeatAlertName: AlertLevel{
					AlertName:  heartbeatAlertName,
					AlertLevel: "high",
				},
			},
		}

		alertLevelP := client.GetAlertLevel(agentName, string(heartbeatAlertName))

		assert.NotNil(t, alertLevelP)
		assert.Equal(t, "high", (*alertLevelP).AlertLevel)
	})

	t.Run("GetAlertLevelTest - happy path multiple", func(t *testing.T) {
		tmEventCommitType := "tm:event:commit"
		compositeAlertName := fmt.Sprintf("%s,%s", heartbeatAlertName, tmEventCommitType)

		client.AgentAlertLevelList = map[AgentName]map[AlertName]AlertLevel{
			agentName: {
				heartbeatAlertName: AlertLevel{
					AlertName:  heartbeatAlertName,
					AlertLevel: "high",
				},
				AlertName(compositeAlertName): {
					AlertName:  AlertName(compositeAlertName),
					AlertLevel: "composite",
				},
			},
		}

		var alertLevel AlertLevel
		alertLevelP := client.GetAlertLevel(agentName, heartbeatAlertName.String(), tmEventCommitType)

		assert.NotNil(t, alertLevelP)
		alertLevel = *alertLevelP
		assert.Equal(t, "composite", alertLevel.AlertLevel)
	})

	t.Run("GetAlertLevelTest - happy path twice", func(t *testing.T) {
		tmEventCommitType := "tm:event:commit"

		client.AgentAlertLevelList = map[AgentName]map[AlertName]AlertLevel{
			agentName: {
				heartbeatAlertName: AlertLevel{
					AlertName:  heartbeatAlertName,
					AlertLevel: "high",
				},
			},
		}

		var alertLevel AlertLevel
		alertLevelP := client.GetAlertLevel(agentName, heartbeatAlertName.String(), tmEventCommitType)
		assert.Nil(t, alertLevelP)
		alertLevelP = client.GetAlertLevel(agentName, heartbeatAlertName.String())
		assert.NotNil(t, alertLevelP)

		alertLevel = *alertLevelP
		assert.Equal(t, "high", alertLevel.AlertLevel)
	})

	t.Run("GetAlertLevelTest - bad path multiple", func(t *testing.T) {
		client.AgentAlertLevelList = map[AgentName]map[AlertName]AlertLevel{
			agentName: {
				heartbeatAlertName: AlertLevel{
					AlertName:  heartbeatAlertName,
					AlertLevel: "high",
				},
			},
		}

		alertLevelP := client.GetAlertLevel(agentName, heartbeatAlertName.String(), "tm:event:commit")

		assert.Nil(t, alertLevelP)
	})

}
