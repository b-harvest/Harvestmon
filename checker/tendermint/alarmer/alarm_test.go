package alarmer

import (
	database "github.com/b-harvest/Harvestmon/database"
	"github.com/stretchr/testify/assert"
	"tendermint-checker/types"
	"testing"
	"time"
)

func Test(t *testing.T) {
	ts := 10 * time.Second
	cfg := types.CheckerConfig{
		CommitId:      "alarm",
		CheckInterval: &ts,
		Database: database.Database{
			User:      "root",
			Password:  "helloworld",
			Host:      "127.0.0.1",
			Port:      33306,
			DbName:    "harvestmon",
			AwsRegion: "",
		},
	}
	t.Run("happy path", func(t *testing.T) {

		client, err := types.NewCheckerClient(&cfg, &types.AlertDefinition{}, []types.CustomAgentConfig{})
		assert.NoError(t, err)

		alert := types.NewAlert(types.Alarmer{
			AlarmerName: "harvestmon-telegram",
			AlarmParamList: map[string]any{
				"chat": 6194601082,
			},
		}, types.AlertLevel{AlertName: "tendermint:test", AlertLevel: "high"}, "[T] jinu.t.kr", "")
		err = RunAlarm(&cfg, *client, alert)

		assert.NoError(t, err)
	})

}
