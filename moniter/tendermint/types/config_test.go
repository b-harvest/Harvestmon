package types

import (
	database "github.com/b-harvest/Harvestmon/database"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
	"time"
)

func Test(t *testing.T) {

	t.Run("config.yaml parsing", func(t *testing.T) {
		var (
			mConfig MonitorConfig
		)

		configBytes := []byte(
			"agent:\n" +
				"  name: \"polkachu.com\"\n" +
				"  host: \"cosmos-rpc.polkachu.com\"\n" +
				"  port: 443\n" +
				"  pushInterval: 10s\n" +
				"  timeout: 10s\n" +
				"  commitId: 19ge4rgndfifji\n" +
				"database:\n" +
				"  user: root\n" +
				"  password: helloWorld\n" +
				"  host: 127.0.0.1\n" +
				"  port: 33306\n" +
				"  dbName: harvestmon")

		err := yaml.Unmarshal(configBytes, &mConfig)
		assert.NoError(t, err)

		err = mConfig.ApplyConfigFromEnvAndDefault()
		assert.NoError(t, err)

		ts := time.Second * 10
		assert.Equal(t, MonitorConfig{
			Agent: MonitoringAgent{
				AgentName:    "polkachu.com",
				Host:         "cosmos-rpc.polkachu.com",
				Port:         443,
				PushInterval: &ts,
				Timeout:      &ts,
				CommitId:     "19ge4rgndfifji",
				Monitors:     nil,
			},
			Database: database.Database{
				User:      "hello",
				Password:  "helloworld",
				Host:      "127.0.0.1",
				Port:      33306,
				DbName:    "harvestmon",
				AwsRegion: "",
			},
		}, mConfig)
	})

}
