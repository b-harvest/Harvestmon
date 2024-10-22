package alarmer

import (
	database "github.com/b-harvest/Harvestmon/database"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func Test(t *testing.T) {
	ts := 10 * time.Second
	cfg := CheckerConfig{
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

		client, err := NewCheckerClient(&cfg, &AlertDefinition{}, []CustomAgentConfig{})
		assert.NoError(t, err)

		alert := NewAlert(Alarmer{
			AlarmerName: "harvestmon-telegram",
			AlarmParamList: map[string]any{
				"chat": 6194601082,
			},
		}, AlertLevel{AlertName: "tendermint:test", AlertLevel: "high"}, "[T] jinu.t.kr", "")
		err = RunAlarm(&cfg, *client, alert)

		assert.NoError(t, err)
	})

}
