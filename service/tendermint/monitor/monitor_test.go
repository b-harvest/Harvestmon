package monitor

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"github.com/b-harvest/Harvestmon/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"net/http"
	"net/http/httptest"
	"strconv"
	"tendermint-mon/types"
	"testing"
	"time"
)

type mockRoundTripper struct {
	response *http.Response
}

func (rt *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return rt.response, nil
}

type MockQueryer struct {
	db sql.DB
}

func (m *MockQueryer) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return driver.RowsAffected(1), nil
}

func (m *MockQueryer) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return nil, nil
}

func Test(t *testing.T) {
	cfg := types.MonitorConfig{
		Agent: types.MonitoringAgent{
			AgentName:    "polkachu.com",
			Host:         "cosmos-rpc.polkachu.com",
			Port:         443,
			PushInterval: time.Second * 10,
			Timeout:      time.Second * 10,
			CommitId:     "19ge4rgndfifji",
			Monitors:     nil,
		},
		Database: types.Database{
			User:      "root",
			Password:  "helloworld",
			Host:      "127.0.0.1",
			Port:      33306,
			DbName:    "harvestmon",
			AwsRegion: "",
		},
	}

	json := `{
  "jsonrpc": "2.0",
  "id": -1,
  "result": {
    "listening": true,
    "listeners": [
      "Listener(@65.108.131.174:14956)"
    ],
    "n_peers": "97",
    "peers": [
      {
        "node_info": {
          "protocol_version": {
            "p2p": "8",
            "block": "11",
            "app": "0"
          },
          "id": "ce345ae23f0d16e5d843c1f84f8e410d732b5bd8",
          "listen_addr": "46.105.71.65:26656",
          "network": "cosmoshub-4",
          "version": "0.37.6",
          "channels": "40202122233038606100",
          "moniker": "CosmoshubTheGreat",
          "other": {
            "tx_index": "off",
            "rpc_address": "tcp://0.0.0.0:26657"
          }
        },
        "is_outbound": false,
        "connection_status": {
          "Duration": "108380833730685",
          "SendMonitor": {
            "Start": "2024-08-03T05:00:06.28Z",
            "Bytes": "1565612157",
            "Samples": "406907",
            "InstRate": "175",
            "CurRate": "12807",
            "AvgRate": "14445",
            "PeakRate": "1684320",
            "BytesRem": "0",
            "Duration": "108380760000000",
            "Idle": "40000000",
            "TimeRem": "0",
            "Progress": 0,
            "Active": true
          },
          "RecvMonitor": {
            "Start": "2024-08-03T05:00:06.28Z",
            "Bytes": "2103965476",
            "Samples": "385706",
            "InstRate": "950",
            "CurRate": "12440",
            "AvgRate": "19413",
            "PeakRate": "1611840",
            "BytesRem": "0",
            "Duration": "108380820000000",
            "Idle": "20000000",
            "TimeRem": "0",
            "Progress": 0,
            "Active": true
          },
          "Channels": [
            {
              "ID": 48,
              "SendQueueCapacity": "1",
              "SendQueueSize": "0",
              "Priority": "5",
              "RecentlySent": "43288"
            },
            {
              "ID": 64,
              "SendQueueCapacity": "1000",
              "SendQueueSize": "0",
              "Priority": "5",
              "RecentlySent": "0"
            },
            {
              "ID": 32,
              "SendQueueCapacity": "100",
              "SendQueueSize": "0",
              "Priority": "6",
              "RecentlySent": "8903"
            },
            {
              "ID": 33,
              "SendQueueCapacity": "100",
              "SendQueueSize": "0",
              "Priority": "10",
              "RecentlySent": "28798"
            },
            {
              "ID": 34,
              "SendQueueCapacity": "100",
              "SendQueueSize": "0",
              "Priority": "7",
              "RecentlySent": "36363"
            },
            {
              "ID": 35,
              "SendQueueCapacity": "2",
              "SendQueueSize": "0",
              "Priority": "1",
              "RecentlySent": "7"
            },
            {
              "ID": 56,
              "SendQueueCapacity": "1",
              "SendQueueSize": "0",
              "Priority": "6",
              "RecentlySent": "0"
            },
            {
              "ID": 96,
              "SendQueueCapacity": "10",
              "SendQueueSize": "0",
              "Priority": "5",
              "RecentlySent": "0"
            },
            {
              "ID": 97,
              "SendQueueCapacity": "10",
              "SendQueueSize": "0",
              "Priority": "3",
              "RecentlySent": "0"
            },
            {
              "ID": 0,
              "SendQueueCapacity": "10",
              "SendQueueSize": "0",
              "Priority": "1",
              "RecentlySent": "0"
            }
          ]
        },
        "remote_ip": "46.105.71.65"
      },
      {
        "node_info": {
          "protocol_version": {
            "p2p": "8",
            "block": "11",
            "app": "0"
          },
          "id": "9755cab2585a2794453a5b396ef13b893393366f",
          "listen_addr": "tcp://0.0.0.0:46671",
          "network": "cosmoshub-4",
          "version": "1.0.0",
          "channels": "00",
          "moniker": "cosmoshub-4-multiseed",
          "other": {
            "tx_index": "",
            "rpc_address": ""
          }
        },
        "is_outbound": false,
        "connection_status": {
          "Duration": "868550244518",
          "SendMonitor": {
            "Start": "2024-08-04T10:51:58.56Z",
            "Bytes": "16341",
            "Samples": "176",
            "InstRate": "0",
            "CurRate": "0",
            "AvgRate": "19",
            "PeakRate": "1719",
            "BytesRem": "0",
            "Duration": "868560000000",
            "Idle": "28560000000",
            "TimeRem": "0",
            "Progress": 0,
            "Active": true
          },
          "RecvMonitor": {
            "Start": "2024-08-04T10:51:58.56Z",
            "Bytes": "93",
            "Samples": "176",
            "InstRate": "0",
            "CurRate": "0",
            "AvgRate": "0",
            "PeakRate": "30",
            "BytesRem": "0",
            "Duration": "868560000000",
            "Idle": "28560000000",
            "TimeRem": "0",
            "Progress": 0,
            "Active": true
          },
          "Channels": [
            {
              "ID": 48,
              "SendQueueCapacity": "1",
              "SendQueueSize": "0",
              "Priority": "5",
              "RecentlySent": "0"
            },
            {
              "ID": 64,
              "SendQueueCapacity": "1000",
              "SendQueueSize": "0",
              "Priority": "5",
              "RecentlySent": "0"
            },
            {
              "ID": 32,
              "SendQueueCapacity": "100",
              "SendQueueSize": "0",
              "Priority": "6",
              "RecentlySent": "0"
            },
            {
              "ID": 33,
              "SendQueueCapacity": "100",
              "SendQueueSize": "0",
              "Priority": "10",
              "RecentlySent": "0"
            },
            {
              "ID": 34,
              "SendQueueCapacity": "100",
              "SendQueueSize": "0",
              "Priority": "7",
              "RecentlySent": "0"
            },
            {
              "ID": 35,
              "SendQueueCapacity": "2",
              "SendQueueSize": "0",
              "Priority": "1",
              "RecentlySent": "0"
            },
            {
              "ID": 56,
              "SendQueueCapacity": "1",
              "SendQueueSize": "0",
              "Priority": "6",
              "RecentlySent": "0"
            },
            {
              "ID": 96,
              "SendQueueCapacity": "10",
              "SendQueueSize": "0",
              "Priority": "5",
              "RecentlySent": "0"
            },
            {
              "ID": 97,
              "SendQueueCapacity": "10",
              "SendQueueSize": "0",
              "Priority": "3",
              "RecentlySent": "0"
            },
            {
              "ID": 0,
              "SendQueueCapacity": "10",
              "SendQueueSize": "0",
              "Priority": "1",
              "RecentlySent": "0"
            }
          ]
        },
        "remote_ip": "65.108.212.224"
      },
      {
        "node_info": {
          "protocol_version": {
            "p2p": "8",
            "block": "11",
            "app": "0"
          },
          "id": "5b4529df65f9c1006d51472a827f1deb23825ba2",
          "listen_addr": "tcp://0.0.0.0:14656",
          "network": "cosmoshub-4",
          "version": "0.37.6",
          "channels": "40202122233038606100",
          "moniker": "composer",
          "other": {
            "tx_index": "on",
            "rpc_address": "tcp://0.0.0.0:14657"
          }
        },
        "is_outbound": false,
        "connection_status": {
          "Duration": "5626122412306",
          "SendMonitor": {
            "Start": "2024-08-04T09:32:40.98Z",
            "Bytes": "101830026",
            "Samples": "20980",
            "InstRate": "175",
            "CurRate": "13339",
            "AvgRate": "18100",
            "PeakRate": "966500",
            "BytesRem": "0",
            "Duration": "5626060000000",
            "Idle": "20000000",
            "TimeRem": "0",
            "Progress": 0,
            "Active": true
          },
          "RecvMonitor": {
            "Start": "2024-08-04T09:32:40.98Z",
            "Bytes": "97128106",
            "Samples": "18462",
            "InstRate": "944",
            "CurRate": "12973",
            "AvgRate": "17264",
            "PeakRate": "867970",
            "BytesRem": "0",
            "Duration": "5626140000000",
            "Idle": "160000000",
            "TimeRem": "0",
            "Progress": 0,
            "Active": true
          },
          "Channels": [
            {
              "ID": 48,
              "SendQueueCapacity": "1",
              "SendQueueSize": "0",
              "Priority": "5",
              "RecentlySent": "42294"
            },
            {
              "ID": 64,
              "SendQueueCapacity": "1000",
              "SendQueueSize": "0",
              "Priority": "5",
              "RecentlySent": "0"
            },
            {
              "ID": 32,
              "SendQueueCapacity": "100",
              "SendQueueSize": "0",
              "Priority": "6",
              "RecentlySent": "7996"
            },
            {
              "ID": 33,
              "SendQueueCapacity": "100",
              "SendQueueSize": "0",
              "Priority": "10",
              "RecentlySent": "37442"
            },
            {
              "ID": 34,
              "SendQueueCapacity": "100",
              "SendQueueSize": "0",
              "Priority": "7",
              "RecentlySent": "64688"
            },
            {
              "ID": 35,
              "SendQueueCapacity": "2",
              "SendQueueSize": "0",
              "Priority": "1",
              "RecentlySent": "16"
            },
            {
              "ID": 56,
              "SendQueueCapacity": "1",
              "SendQueueSize": "0",
              "Priority": "6",
              "RecentlySent": "0"
            },
            {
              "ID": 96,
              "SendQueueCapacity": "10",
              "SendQueueSize": "0",
              "Priority": "5",
              "RecentlySent": "0"
            },
            {
              "ID": 97,
              "SendQueueCapacity": "10",
              "SendQueueSize": "0",
              "Priority": "3",
              "RecentlySent": "0"
            },
            {
              "ID": 0,
              "SendQueueCapacity": "10",
              "SendQueueSize": "0",
              "Priority": "1",
              "RecentlySent": "0"
            }
          ]
        },
        "remote_ip": "89.33.22.102"
      }
    ]
  }
}`

	t.Run("get net_info", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		recorder.Header().Add("Content-Type", "application/json")
		_, err := recorder.WriteString(json)
		assert.NoError(t, err)

		expectedResponse := recorder.Result()
		client := types.NewMonitorClient(&cfg, &http.Client{Transport: &mockRoundTripper{response: expectedResponse}})

		netInfo, err := client.GetNetInfo()
		assert.NoError(t, err)

		assert.Equal(t,
			"5b4529df65f9c1006d51472a827f1deb23825ba2",
			string(netInfo.Result.Peers[len(netInfo.Result.Peers)-1].NodeInfo.DefaultNodeID))
	})

	json = `{
  "jsonrpc": "2.0",
  "id": -1,
  "result": {
    "signed_header": {
      "header": {
        "version": {
          "block": "11"
        },
        "chain_id": "axelar-dojo-1",
        "height": "13875975",
        "time": "2024-08-04T12:19:06.306115453Z",
        "last_block_id": {
          "hash": "EF62FB0202E75311E123681C0EFA3D39895AABA5EF148807562BF0AAB20ADE04",
          "parts": {
            "total": 1,
            "hash": "FC72BD454BAEBE8D06C89307C5C9BAB2337F9E141CE0C4FF73AD60E9B50CBFA2"
          }
        },
        "last_commit_hash": "522AD2A839E2BCC742C7007FC9EF87D03E1AC6E7067D645BAAA87D88FF63F1F3",
        "data_hash": "9FA2F296DD9BD65734A667283780A8986703A09665725CA895173BFFEAB2C045",
        "validators_hash": "7E98AAED78450716B249EA0A17CA1FD60CAE57675DCDA0A49D4089E2985B748A",
        "next_validators_hash": "7E98AAED78450716B249EA0A17CA1FD60CAE57675DCDA0A49D4089E2985B748A",
        "consensus_hash": "048091BC7DDC283F77BFBF91D73C44DA58C3DF8A9CBC867405D8B7F3DAADA22F",
        "app_hash": "B5AAB98FDA526407A9DBF6543C622B6FB3F114F398ED175FDF3A63362A0C4FBB",
        "last_results_hash": "CCD0E34103C7D2196290315566A9172174DC0461C08C3E941314FD7F33C19805",
        "evidence_hash": "E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855",
        "proposer_address": "0B7EA0F411F506695D41F188343B3FB5F4CF789C"
      },
      "commit": {
        "height": "13875975",
        "round": 0,
        "block_id": {
          "hash": "9DABDFFADDC911C71C226F8E44E6AA9F5D7EBA13F5DAE857EF7B93477B7E2063",
          "parts": {
            "total": 1,
            "hash": "F3CE8BFD29C966BD6422F39366826941C34C6234F3116C1E23883ADF4DDDB776"
          }
        },
        "signatures": [
          {
            "block_id_flag": 2,
            "validator_address": "3AAE040478D0B916EAE5B2EAE4B2C39D5F1EFCE6",
            "timestamp": "2024-08-04T12:19:12.142391573Z",
            "signature": "BneOSHPGlTmvzrk+xuoZnKvOGh4q2QDEsMsQfB4dVPMqOh4bgAJMuY8RogD1gdmnvk28e4+BO4S1jwcx5V6JDQ=="
          },
          {
            "block_id_flag": 2,
            "validator_address": "2CE83523458CC4B4ABD59A53F89744947573677E",
            "timestamp": "2024-08-04T12:19:12.157518928Z",
            "signature": "rv1ye3jdddU+Jsqusq9pGnWajm3HvH8qNvCBUKywVNXOiiUqYNGfngJAFj1GqkFsdlyHhjrFIAztzeSvPlsPDg=="
          },
          {
            "block_id_flag": 1,
            "validator_address": "",
            "timestamp": "0001-01-01T00:00:00Z",
            "signature": null
          },
          {
            "block_id_flag": 2,
            "validator_address": "B0F07A73B0402316F406EDFAF8863CCB140C88B4",
            "timestamp": "2024-08-04T12:19:12.14733235Z",
            "signature": "jo+iDz7vLcrvxj5diemXviddmJIxNv6/siXS8CYLHMUINukLKJlJcuIL1gZKW6khPUeusT22rGTZ/+KngSniDQ=="
          },
          {
            "block_id_flag": 2,
            "validator_address": "C8E7FA98EAF061E786393C94AE29A5831EDE363E",
            "timestamp": "2024-08-04T12:19:12.152243317Z",
            "signature": "RJI0Wio6R4BUsOhzn5O+qyRhvLqLgi44i9dvn/f2et+JnN/KaLEKpLVObeuxrxg8LY1LRV6XrT1VFiKMG3TwCA=="
          },
          {
            "block_id_flag": 1,
            "validator_address": "",
            "timestamp": "0001-01-01T00:00:00Z",
            "signature": null
          },
          {
            "block_id_flag": 2,
            "validator_address": "D82FB64AC8E00DAB4E6AC6D9370023A2281B7256",
            "timestamp": "2024-08-04T12:19:12.182352671Z",
            "signature": "Opyr4zyjn2/ER/fTaiJVGSDnyIQbiPjElNHNToMY+/apdFLPVclp6VP/ISDpPl/CLmE5Wox+e3n+CBB6wXzYCQ=="
          },
          {
            "block_id_flag": 1,
            "validator_address": "",
            "timestamp": "0001-01-01T00:00:00Z",
            "signature": null
          },
          {
            "block_id_flag": 2,
            "validator_address": "4C3EAD0A99B1B7F7AF9C45D41B0A7F7D7CE2D67B",
            "timestamp": "2024-08-04T12:19:12.130735258Z",
            "signature": "6yMJHltGQSFPk8dTDQ5jRqwRg8kUpnxT+JFoJGKxbRIBsCVGF7jA2bakfDr5EERS2FnD18pQKCZo60dPTuwLBA=="
          },
          {
            "block_id_flag": 2,
            "validator_address": "8C5872749B2182B1832D3EAE57EDFAC01104B240",
            "timestamp": "2024-08-04T12:19:12.146689573Z",
            "signature": "7+dioagOv1dc7y3+EIor7GTIACw/F9Mryf86+nLIxwkuc/h/UMiShmcMNpVqBFAQ9Jtm8m6IW1d7ljdXKry9AQ=="
          },
          {
            "block_id_flag": 1,
            "validator_address": "",
            "timestamp": "0001-01-01T00:00:00Z",
            "signature": null
          },
          {
            "block_id_flag": 2,
            "validator_address": "9AA1BD7F54997CF32113104393DCEA7BC3BCE836",
            "timestamp": "2024-08-04T12:19:12.19789524Z",
            "signature": "xS4S3zUG5H1fNtXjMzzsN4n+trCYzVA3N2s4DERDoZ5PXGgBMnHIHYIiQkkHpn+vUflRanyf4y5PRQw86MhWDA=="
          },
          {
            "block_id_flag": 2,
            "validator_address": "D94070533C76A164E37C0FA62A845A427CD34E40",
            "timestamp": "2024-08-04T12:19:12.114747554Z",
            "signature": "eFVBtnUWXmvEy2GTFW5tRuNJH77HKjN0W+amwTlRa75aBNJKxZNh7cfckW/fNCr6NesTw3+eRjsrvuKlPnH+BQ=="
          },
          {
            "block_id_flag": 1,
            "validator_address": "",
            "timestamp": "0001-01-01T00:00:00Z",
            "signature": null
          },
          {
            "block_id_flag": 2,
            "validator_address": "8F3695E65CBFB6D9084AFC0B637935E97CB3A381",
            "timestamp": "2024-08-04T12:19:12.222397819Z",
            "signature": "xwjpWRB6YeiOTHgSFQaYOYNT6MkburDZHPShFhcmz1Blx6nEMKJuJfdhXx1y8FzNwyTrr6628mjPcn677qyBDQ=="
          },
          {
            "block_id_flag": 1,
            "validator_address": "",
            "timestamp": "0001-01-01T00:00:00Z",
            "signature": null
          },
          {
            "block_id_flag": 1,
            "validator_address": "",
            "timestamp": "0001-01-01T00:00:00Z",
            "signature": null
          },
          {
            "block_id_flag": 2,
            "validator_address": "48181C41E43158F0ADE8011C86CF715BE87BD7F4",
            "timestamp": "2024-08-04T12:19:12.117245201Z",
            "signature": "ST7mWHur4ij6Ppp6GEdRzKbojHhhHqPjPwnLwRvNIL3KP3sCu2ZKtGWHuLwPhRoRkX00pfB8+DU+VIvmQHFmAA=="
          },
          {
            "block_id_flag": 2,
            "validator_address": "A5ACE1813423D1AB76E9BD17264945B5A9F4EB06",
            "timestamp": "2024-08-04T12:19:12.099612727Z",
            "signature": "VsnXrg0Xfrd0CiHLgiONWVZIPSxngsfrkiKOrsqvpidDdR389i8MX1gYjTeNyTN0uhIjCMpVpD6oH7gXXkS8CA=="
          },
          {
            "block_id_flag": 2,
            "validator_address": "4338C4CDE85BFCF4F04E2ECBCBF7007DD93185FA",
            "timestamp": "2024-08-04T12:19:12.134898938Z",
            "signature": "MiaGf1/3kdZf72uvagem36TQs3ZGd2eBY9gqvASmT3ojTRCvVutz951WvytGRq/1Ujc3H1UT3Qc2kgfqOP6dCA=="
          },
          {
            "block_id_flag": 2,
            "validator_address": "B83975807B9428E001FEBC6449FD2267F0E50DFF",
            "timestamp": "2024-08-04T12:19:12.120942597Z",
            "signature": "Pb7wWf/8RxRc9ZVp9Ujj/6g5hSedkNRXTZlG9pMLAwnUwn9zsFcyKJYR80lygb4D3tk2xb/lXrShuPiI6Mw3Ag=="
          },
          {
            "block_id_flag": 2,
            "validator_address": "5526E315D0C8F78B9DE7A6E60F7D518B91CE023C",
            "timestamp": "2024-08-04T12:19:12.11453517Z",
            "signature": "bfDAW/CvvAxQHF/4ydAqKsQNga6E8K3r2YVo2bBVvYSiISGVdqVm1Fa5AkcFH/8mNBFW+QpooQYOTulPZrHlBQ=="
          },
          {
            "block_id_flag": 1,
            "validator_address": "",
            "timestamp": "0001-01-01T00:00:00Z",
            "signature": null
          },
          {
            "block_id_flag": 2,
            "validator_address": "B286F6A0BD3CD35B1940860A4D912C32FC2D1863",
            "timestamp": "2024-08-04T12:19:12.171525968Z",
            "signature": "Q0KYbA8tDRnwT44Yx6NE/8+fG2kkocRkPfftLWtzOv3NB47Ob9n59P+Fq5oXq9Ianz6oSUBDVvj7LBSap6aIAA=="
          },
          {
            "block_id_flag": 1,
            "validator_address": "",
            "timestamp": "0001-01-01T00:00:00Z",
            "signature": null
          },
          {
            "block_id_flag": 2,
            "validator_address": "7EC661F40BEFD355FEF828AF6462E574F4759D0B",
            "timestamp": "2024-08-04T12:19:12.157197887Z",
            "signature": "16W+C1frs+X6wekTDZY3W118yF66WlWhoWPfsk3ZJ1UifwZxVAmTVnsI+p/Th2/HgkbOMBAD886GJVRbLu+dAg=="
          },
          {
            "block_id_flag": 1,
            "validator_address": "",
            "timestamp": "0001-01-01T00:00:00Z",
            "signature": null
          },
          {
            "block_id_flag": 2,
            "validator_address": "8740C9E2428E2D8B5B2002552E0D2880EC6E1AB0",
            "timestamp": "2024-08-04T12:19:12.117686646Z",
            "signature": "zGB0ahB+Kv8PmZIdy0ES13gBPSSIwDHTU/0IW8UNN20+FIPbDQ/7rKP5HJ0rByq3Uzu70kA8LUVY+RTO74SRCQ=="
          },
          {
            "block_id_flag": 2,
            "validator_address": "6A6D5C2FA9CE7DE300D89C703132681D69155131",
            "timestamp": "2024-08-04T12:19:12.135594183Z",
            "signature": "4P5SE1EUkjRUHI8ccdASL7NlGoV3CWQNSZPR1fnAejV5KHep3v2IxEIha/fQypH3cszzaE5zEZGJ46z+ohCNCw=="
          },
          {
            "block_id_flag": 2,
            "validator_address": "080917AD1384CD7FC87B22ECACA1010CA397C363",
            "timestamp": "2024-08-04T12:19:12.117989262Z",
            "signature": "WYMlRIsGUIqilCBn0BHGUWpVp1U7aPFwKxnPTW2TygJV+m+peozA4arhdY59XTNPQafpS6ZOOitwILpXVXAtCg=="
          },
          {
            "block_id_flag": 2,
            "validator_address": "030EACE62A0FDC87356F9E2BE2C8EB2D8376F272",
            "timestamp": "2024-08-04T12:19:12.12357028Z",
            "signature": "8/sscKK9ouL0uMZTOrIxk/K5Ja6ekJE9bDF9EKEpIoWoVg65ip6/2tOodeVzYbmEdKAXXkzS5l09eBvtuyIuDw=="
          },
          {
            "block_id_flag": 1,
            "validator_address": "",
            "timestamp": "0001-01-01T00:00:00Z",
            "signature": null
          },
          {
            "block_id_flag": 2,
            "validator_address": "97C2453953CB1472E9C3C629EE58115055EB2793",
            "timestamp": "2024-08-04T12:19:12.160190473Z",
            "signature": "6Q1pbQV/ARAZ7ky01lgT7NoxVSrV82qp4+GwJRoZmHsUwpo+qCd97QX0OF91eBVHq4LvPvLa79ct6zGTKJjBBg=="
          },
          {
            "block_id_flag": 2,
            "validator_address": "9962209CBF2BB908DC5CAF4B608AC75E7701BB6C",
            "timestamp": "2024-08-04T12:19:12.15524508Z",
            "signature": "HJ7d6eKTd+S3YK7IQIDwv0Srb3PxVVcskXS+h+WGjGTEnrrkTni51mf0FNczKktKsspfUm7rOGrrj/pOeFoaDg=="
          },
          {
            "block_id_flag": 2,
            "validator_address": "B3AC06F1CFAE7748FBAB70BE762855762A44227A",
            "timestamp": "2024-08-04T12:19:12.13777464Z",
            "signature": "5bilf1aWVfp+J1Tf18IM4Tt2R0mX+/L6+qKWkahFO65DaIipvqvu4kmdPhplGBE3d2IizVRdmnVPz5DmDIHhAA=="
          },
          {
            "block_id_flag": 2,
            "validator_address": "3197C8328BCF200D00AC28909BEB99BAE57F7263",
            "timestamp": "2024-08-04T12:19:12.147746Z",
            "signature": "nCmpfyUEVK6dDJhxpxmTiPOY7mDLMAMkNXKsUoxbfToOkP6y38ZdeBxXopdUybbORZMAzbPjDDKktMyvr27ZCw=="
          },
          {
            "block_id_flag": 2,
            "validator_address": "CA1086B803C8AC4E754CCF1842B8EAA8ADC17292",
            "timestamp": "2024-08-04T12:19:12.192635476Z",
            "signature": "RA2EYQamin9OjpegKALwH7WqqP73v1iVtBNqrf/2417TD6tagU8V6lTOHjlu6PnlJwD4is2V2vJuf1XXASmHCg=="
          },
          {
            "block_id_flag": 2,
            "validator_address": "257665698E33ADA42087491F898346FEBAE27E1F",
            "timestamp": "2024-08-04T12:19:12.134173795Z",
            "signature": "7ujq2dqJSGM56JwMprtrEaO/p6mGYc+XWMkSI+MrzC1YJ5E32dhPu8axgwzkGbi4iFs+ss+idruHLpgD0HT4Bw=="
          },
          {
            "block_id_flag": 2,
            "validator_address": "518D2BF6CACD34AF9A797B442090168177FF4850",
            "timestamp": "2024-08-04T12:19:12.11382467Z",
            "signature": "4WFpIgj4pm4DW6jwUa/v0EgJ8pZUa3nYP6+l6C51q/Gkc8TkXGPbpc9HMqJ6OknDvh88ybGuFlXx38KHEKMaBg=="
          },
          {
            "block_id_flag": 1,
            "validator_address": "",
            "timestamp": "0001-01-01T00:00:00Z",
            "signature": null
          },
          {
            "block_id_flag": 1,
            "validator_address": "",
            "timestamp": "0001-01-01T00:00:00Z",
            "signature": null
          },
          {
            "block_id_flag": 1,
            "validator_address": "",
            "timestamp": "0001-01-01T00:00:00Z",
            "signature": null
          },
          {
            "block_id_flag": 2,
            "validator_address": "C3F0F48DA2EC5B3E263F18146114FD6DBB37755B",
            "timestamp": "2024-08-04T12:19:12.138824772Z",
            "signature": "2lqn+avx9AmgwluBN1RcG7cEXgF40jDRwgIM753wUrwaDy5S2yYzr8xNRdrOpNEy0ZFr5zEJzuZllHFZXOLLAA=="
          },
          {
            "block_id_flag": 2,
            "validator_address": "81DA03EFE16079C6197C2949D3F82F835E19B90C",
            "timestamp": "2024-08-04T12:19:12.124017099Z",
            "signature": "rRbZ4i4mMdFgvYApyvrnDtlbPXwQqBq4VLFWUV9Aq2qXYqa8BKwFGKq1lKL9PhNUL6nHbZwtjHp0lfzC216nAw=="
          },
          {
            "block_id_flag": 2,
            "validator_address": "F9B4D9313C569F39DF1F3C664B55648F13F09D2A",
            "timestamp": "2024-08-04T12:19:12.19065845Z",
            "signature": "2MvepAMYHQvyKW4OIHtyMbO0WNQHwJ98Dh8rVH7IuAfYUMYZaRyST2LYKWXCX6/IMSiQPwEAzkuNEoBQMTZAAQ=="
          },
          {
            "block_id_flag": 1,
            "validator_address": "",
            "timestamp": "0001-01-01T00:00:00Z",
            "signature": null
          },
          {
            "block_id_flag": 1,
            "validator_address": "",
            "timestamp": "0001-01-01T00:00:00Z",
            "signature": null
          },
          {
            "block_id_flag": 2,
            "validator_address": "FC8D54A406146FC0E7C6349E6D54E0929DAF9B2C",
            "timestamp": "2024-08-04T12:19:12.146290742Z",
            "signature": "NCyJ8WOwHvKTuXZzzSPI7seBYRpp0aov4n9FblHztJ/RHBLJErBVdpiQe2WmJ661OD5tjaezceiKbBOw+PaMDw=="
          },
          {
            "block_id_flag": 2,
            "validator_address": "6FEE4C6B4CF639B76357863F5B2DC35587BCBB60",
            "timestamp": "2024-08-04T12:19:12.155803318Z",
            "signature": "is73RwaJ9bU8knZ0d7OEMjFTfw5cVVggnQELEXDI5nuwOARjOBtP9xXJEUYmco3zmTwgNiNscs2+nuGo2aawCA=="
          },
          {
            "block_id_flag": 2,
            "validator_address": "2E333AEB99DEE073AB9DCE544A2B63384FE3EB38",
            "timestamp": "2024-08-04T12:19:12.111881518Z",
            "signature": "UegTdxlRmCkSPpye0VrfmmeJD4zyCpEe4oD1xwjq2qQwGXqm9zeLM15H9cKUUH7uxkHAJelNeGkurf4IzGYqCw=="
          },
          {
            "block_id_flag": 1,
            "validator_address": "",
            "timestamp": "0001-01-01T00:00:00Z",
            "signature": null
          },
          {
            "block_id_flag": 2,
            "validator_address": "85F3D82818E7BA6C639CB094664326F5001CF3FA",
            "timestamp": "2024-08-04T12:19:12.124881368Z",
            "signature": "W7heBg4heW+ysQ2hxC39Qo1IO3bMoyglhn5DqbwWXI/1o/61Vw3iGM+rC+gU3q9eJoZ4vCj+Rwu7jb35xvBPBQ=="
          },
          {
            "block_id_flag": 1,
            "validator_address": "",
            "timestamp": "0001-01-01T00:00:00Z",
            "signature": null
          },
          {
            "block_id_flag": 2,
            "validator_address": "A8CB49A70B9E6539BAB773F5F2CB33C6B90F04E1",
            "timestamp": "2024-08-04T12:19:12.135131861Z",
            "signature": "H9oS9qrJWXrx6Y9l7+1VPGG34rxnmpYAJxxVuLTX5QeHZIzyARrhrzlNm2kHbJmYG3haY8D+p4pntNqjqZDjDQ=="
          },
          {
            "block_id_flag": 1,
            "validator_address": "",
            "timestamp": "0001-01-01T00:00:00Z",
            "signature": null
          },
          {
            "block_id_flag": 1,
            "validator_address": "",
            "timestamp": "0001-01-01T00:00:00Z",
            "signature": null
          },
          {
            "block_id_flag": 2,
            "validator_address": "22CBF3A0B87D9C29BA415DA760BA085214E982A5",
            "timestamp": "2024-08-04T12:19:12.105998756Z",
            "signature": "afFOND/p6S7asZg7W+h6jbiDhQw9W1zi1ncqZzEJxsURTw0mBgQOnKr0rv+0aD7aiIWpzooyqTQJDBQpi8WaBQ=="
          },
          {
            "block_id_flag": 2,
            "validator_address": "779E0D0690544828BAE8756036154E1C4659605A",
            "timestamp": "2024-08-04T12:19:12.134002987Z",
            "signature": "lyBUhzt9MFHut0oSTNH4/XpvGqCLmasSryvDYFODb+CQQPQU6422mMv5if4BbXF+D8xpJSIIfZkguMJ0ci9RDg=="
          },
          {
            "block_id_flag": 2,
            "validator_address": "721ABED113FBE067ADEB535370366254C49C3336",
            "timestamp": "2024-08-04T12:19:12.096795332Z",
            "signature": "30UKKPxqfzw+KsyiVjBWL1Q/9kTQl+GO5diAN4rTVhvx2WNDkoJpW31bOmuyalYjA83CmRZNP3GxLzOyY7NEAA=="
          },
          {
            "block_id_flag": 1,
            "validator_address": "",
            "timestamp": "0001-01-01T00:00:00Z",
            "signature": null
          },
          {
            "block_id_flag": 2,
            "validator_address": "44131EEBD2586953580F02D32BCA8BAA46F3DA61",
            "timestamp": "2024-08-04T12:19:12.12389122Z",
            "signature": "UDBah737YHWLlA/aa4Aul8TQbiM9jnecQQArGfsCcRTuhnr6uPx3224KIDMpfVbdrb/I3AF01Qhc5kmG4e3wDg=="
          },
          {
            "block_id_flag": 2,
            "validator_address": "C687D59DEB9A963364A8970C9F59A02E8ED637A6",
            "timestamp": "2024-08-04T12:19:12.144823899Z",
            "signature": "F9Rltz/83BvWjHgoSKXPOHZJzVIZ412TFxTanAgFzC+N+fsw6A1faoJ0wjziDQmNOkZfnlSvnjG59hiIV+VHCw=="
          },
          {
            "block_id_flag": 1,
            "validator_address": "",
            "timestamp": "0001-01-01T00:00:00Z",
            "signature": null
          },
          {
            "block_id_flag": 1,
            "validator_address": "",
            "timestamp": "0001-01-01T00:00:00Z",
            "signature": null
          },
          {
            "block_id_flag": 2,
            "validator_address": "20DA32578EC9F1BBDF8F0FF9A53543D74970EF93",
            "timestamp": "2024-08-04T12:19:12.158764193Z",
            "signature": "JsNz7Qs2dHuvtdi3f4w7LuBNI7a7q3jSuKOZ0XVI1YCmCGTAyH1WNY5rorTJ3pVb0rSdlfqXTdZRl9UYdU0rCg=="
          },
          {
            "block_id_flag": 2,
            "validator_address": "687058BB56F70B749309B7B0A93A5D653DE5D94B",
            "timestamp": "2024-08-04T12:19:12.162814583Z",
            "signature": "iWa3C3j0u+lhlfpiDkmVKxy2iGdPbt6VzsqzwrevkwHHNNdFyz4r9m+7Z92ooRvNivM+/kikb0NzKOGsxhzSCw=="
          },
          {
            "block_id_flag": 1,
            "validator_address": "",
            "timestamp": "0001-01-01T00:00:00Z",
            "signature": null
          },
          {
            "block_id_flag": 2,
            "validator_address": "27F2AF66185B2DDA1E1D84E31D1746992DF5373A",
            "timestamp": "2024-08-04T12:19:12.118557087Z",
            "signature": "7sTj55eHsGXQmShaCD+psO0k2lJELB75nDxMg9TzgpTKIMM8IavGLiagraZRmc2beEmd17X55wqyU0s9m8TaDQ=="
          },
          {
            "block_id_flag": 2,
            "validator_address": "3EB507B45A3E2311F30AD570D12688380BFCDA8F",
            "timestamp": "2024-08-04T12:19:12.141140502Z",
            "signature": "7x6QfMgkJ7d3etJiJXnyuslwzoZAbXQq8eZYqJ3aZ0SH7yB15LqpFjvRv2pjbb828cr0z0G4u70FO4wr8PwuCQ=="
          },
          {
            "block_id_flag": 1,
            "validator_address": "",
            "timestamp": "0001-01-01T00:00:00Z",
            "signature": null
          },
          {
            "block_id_flag": 1,
            "validator_address": "",
            "timestamp": "0001-01-01T00:00:00Z",
            "signature": null
          },
          {
            "block_id_flag": 2,
            "validator_address": "24C6EA6A9E7EDBD36B0763D5CA792CCC0296F89A",
            "timestamp": "2024-08-04T12:19:12.130773958Z",
            "signature": "oxfSJPyA9u8m26osQCD6dS5DCjOn//xY4i0Qd+zUU94B1S6PY6thc+dFR1AQf1xWqU2YF7IpqCNkGspLinWuAw=="
          },
          {
            "block_id_flag": 2,
            "validator_address": "F6FEE2366243A31AB64B1DC9BE541DDB1D7C290D",
            "timestamp": "2024-08-04T12:19:12.161992178Z",
            "signature": "876yOS4N/XIjRc8XhDHE4giNnfaUeXL63olONYjFIyi9Ar9q/jhNWv1L4guyV8+SKqS6siM7osqIMYb1ZkmeAQ=="
          },
          {
            "block_id_flag": 2,
            "validator_address": "A31C18C4408ADA27DC9AFD0881586AFBC8902A75",
            "timestamp": "2024-08-04T12:19:12.206780233Z",
            "signature": "pZ+u654uEkx4Sqg9MvCOYZUI10nxYV+AtMx2bIIylLZdQRNe/t4fq4r7NgMWZG8Iz3T9a+BZBLwBH+fwlsEtDQ=="
          },
          {
            "block_id_flag": 2,
            "validator_address": "6421F7BA3E55CE4B15DF906276DC3F10F757D763",
            "timestamp": "2024-08-04T12:19:12.160130432Z",
            "signature": "4Y005RRHwcAelLzolg2clFBh6+UE9IO9ufTq9eqeJSXETWzES77GKoTp/PHqxZxbaeFFFZnkuIVP6Qrnsb7RAg=="
          }
        ]
      }
    },
    "canonical": false
  }
}`

	t.Run("get Commit", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		recorder.Header().Add("Content-Type", "application/json")
		_, err := recorder.WriteString(json)
		assert.NoError(t, err)
		expectedResponse := recorder.Result()
		client := types.NewMonitorClient(&cfg, &http.Client{Transport: &mockRoundTripper{response: expectedResponse}})

		commit, err := client.GetCommit()
		assert.NoError(t, err)

		assert.Equal(t,
			"6421F7BA3E55CE4B15DF906276DC3F10F757D763",
			commit.Result.Commit.Signatures[len(commit.Result.Commit.Signatures)-1].ValidatorAddress)
	})

	json = `{
  "jsonrpc": "2.0",
  "id": -1,
  "result": {
    "node_info": {
      "protocol_version": {
        "p2p": "8",
        "block": "11",
        "app": "0"
      },
      "id": "3aa86f390e71f416f66dcf68b22b1b640f1fa23d",
      "listen_addr": "65.108.131.174:14956",
      "network": "cosmoshub-4",
      "version": "0.37.6",
      "channels": "40202122233038606100",
      "moniker": "hello-cosmos-relayer",
      "other": {
        "tx_index": "on",
        "rpc_address": "tcp://0.0.0.0:14957"
      }
    },
    "sync_info": {
      "latest_block_hash": "1C311BE03902E66A00937058762F89FC40EF4420D61653E7E8F565B3E08794AC",
      "latest_app_hash": "ACA060AF848C75A0DFF6798946E953D77C0D1C83353CAAB77AB2BCB4F0064AB2",
      "latest_block_height": "21584725",
      "latest_block_time": "2024-08-04T06:59:09.812577185Z",
      "earliest_block_hash": "151D57031C8E7CE8174F325FE29089BA9D6BA3019D2640B1F4525E6F618A954B",
      "earliest_app_hash": "CA2B8D831B9EE5CFD6F8EDF24F8AEADF4D39DFA8E497280CF5D5DA54FAFE1A15",
      "earliest_block_height": "20685604",
      "earliest_block_time": "2024-06-01T21:59:47.837697303Z",
      "catching_up": false
    },
    "validator_info": {
      "address": "2E172FCAA29C843A518A2A9950763A68351A9075",
      "pub_key": {
        "type": "tendermint/PubKeyEd25519",
        "value": "h6DLniDViaHdmO8G7KfcKXkF4CyznQW8bUdUTA+cf+I="
      },
      "voting_power": "0"
    }
  }
}`

	agent := types.MonitoringAgent{AgentName: "hello", CommitId: "GOOD COMMIT"}

	t.Run("get CometBFT status", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		recorder.Header().Add("Content-Type", "application/json")
		_, err := recorder.WriteString(json)
		assert.NoError(t, err)
		expectedResponse := recorder.Result()
		client := types.NewMonitorClient(&cfg, &http.Client{Transport: &mockRoundTripper{response: expectedResponse}})

		cometBFTStatus, err := client.GetCometBFTStatus()
		assert.NoError(t, err)

		assert.Equal(t,
			types.HexBytes("2E172FCAA29C843A518A2A9950763A68351A9075"),
			cometBFTStatus.ValidatorInfo.Address)
	})

	t.Run("save status", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		recorder.Header().Add("Content-Type", "application/json")
		_, err := recorder.WriteString(json)
		if err != nil {
			return
		}
		expectedResponse := recorder.Result()
		client := types.NewMonitorClient(&cfg, &http.Client{Transport: &mockRoundTripper{response: expectedResponse}})

		statusMonitorRepository := repository.StatusRepository{EventRepository: repository.EventRepository{DB: gorm.DB{}}}

		cometBFTStatus, err := client.GetCometBFTStatus()
		assert.NoError(t, err)

		eventUUID, err := uuid.NewUUID()
		assert.NoError(t, err)

		nodeInfoUUID, err := uuid.NewUUID()
		assert.NoError(t, err)

		createdAt := time.Now()

		latestBlockHeight, err := strconv.ParseUint(cometBFTStatus.SyncInfo.LatestBlockHeight, 0, 64)
		earliestBlockHeight, err := strconv.ParseUint(cometBFTStatus.SyncInfo.EarliestBlockHeight, 0, 64)
		assert.NoError(t, err)

		err = statusMonitorRepository.Save(
			repository.TendermintStatus{
				CreatedAt: createdAt,
				EventUUID: eventUUID.String(),
				Event: repository.Event{
					EventUUID:   eventUUID.String(),
					AgentName:   agent.AgentName,
					ServiceName: types.HARVEST_SERVICE_NAME,
					CommitID:    agent.CommitId,
					EventType:   types.TM_STATUS_EVENT_TYPE,
					CreatedAt:   createdAt,
				},
				TendermintNodeInfoUUID: nodeInfoUUID.String(),
				TendermintNodeInfo: repository.TendermintNodeInfo{
					TendermintNodeInfoUUID: nodeInfoUUID.String(),
					NodeId:                 string(cometBFTStatus.NodeInfo.DefaultNodeID),
					ListenAddr:             cometBFTStatus.NodeInfo.ListenAddr,
					ChainId:                cometBFTStatus.NodeInfo.Network,
					Moniker:                cometBFTStatus.NodeInfo.Moniker,
				},
				LatestBlockHash:     string(cometBFTStatus.SyncInfo.LatestBlockHash),
				LatestAppHash:       string(cometBFTStatus.SyncInfo.LatestAppHash),
				LatestBlockHeight:   latestBlockHeight,
				LatestBlockTime:     cometBFTStatus.SyncInfo.LatestBlockTime,
				EarliestBlockHash:   string(cometBFTStatus.SyncInfo.EarliestBlockHash),
				EarliestAppHash:     string(cometBFTStatus.SyncInfo.EarliestAppHash),
				EarliestBlockHeight: earliestBlockHeight,
				EarliestBlockTime:   cometBFTStatus.SyncInfo.EarliestBlockTime,
				CatchingUp:          cometBFTStatus.SyncInfo.CatchingUp,
			})
		assert.NoError(t, err)
	})
}
