package checker

import (
	"fmt"
	"github.com/b-harvest/Harvestmon/checker/tendermint/types"
)

const (
	ETH_ALARM_TYPE           types.AlertName = "ethereum"
	HEARTBEAT_ETH_ALARM_TYPE types.AlertName = ETH_ALARM_TYPE + ":heartbeat"
	STUCK_ETH_ALARM_TYPE     types.AlertName = ETH_ALARM_TYPE + ":stuck"
)

func blockNumberFormatf(str string, args ...any) string {
	return fmt.Sprintf("[blockNumber] "+str, args...)
}

func heartbeatFormatf(str string, args ...any) string {
	return fmt.Sprintf("[heartbeat] "+str, args...)
}