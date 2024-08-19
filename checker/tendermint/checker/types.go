package checker

import (
	"fmt"
	"github.com/b-harvest/Harvestmon/checker/tendermint/types"
)

const (
	TM_ALARM_TYPE               types.AlertName = "tendermint"
	HEARTBEAT_TM_ALARM_TYPE     types.AlertName = TM_ALARM_TYPE + ":heartbeat"
	HEIGHT_STUCK_TM_ALARM_TYPE  types.AlertName = TM_ALARM_TYPE + ":height_stuck"
	LOW_PEER_TM_ALARM_TYPE      types.AlertName = TM_ALARM_TYPE + ":low_peer"
	MISSING_BLOCK_TM_ALARM_TYPE types.AlertName = TM_ALARM_TYPE + ":missing_block"
)

func netInfoFormatf(str string, args ...any) string {
	return fmt.Sprintf("[net_info] "+str, args...)
}

func blockCommitFormatf(str string, args ...any) string {
	return fmt.Sprintf("[block_commit] "+str, args...)
}

func heartbeatFormatf(str string, args ...any) string {
	return fmt.Sprintf("[heartbeat] "+str, args...)
}

func heightCheckFormatf(str string, args ...any) string {
	return fmt.Sprintf("[height_check] "+str, args...)
}
