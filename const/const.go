package _const

const (
	HARVESTMON_TENDERMINT_SERVICE_NAME = "tendermint"
	TM_EVENT_TYPE                      = "tm:event"
	TM_STATUS_EVENT_TYPE               = TM_EVENT_TYPE + ":status"
	TM_NET_INFO_EVENT_TYPE             = TM_EVENT_TYPE + ":net_info"
	TM_COMMIT_EVENT_TYPE               = TM_EVENT_TYPE + ":commit"

	HARVESTMON_ETHEREUM_SERVICE_NAME = "ethereum"
	ETH_EVENT_TYPE                   = "eth:event"
	ETH_BLOCK_NUMBER_EVENT_TYPE      = ETH_EVENT_TYPE + ":block_number"
	ETH_IS_SYNCING_EVENT_TYPE        = ETH_EVENT_TYPE + ":is_syncing"
)
