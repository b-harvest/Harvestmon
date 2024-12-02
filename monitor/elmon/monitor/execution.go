package monitor

import (
	"errors"
	"fmt"
	// _const "github.com/b-harvest/Harvestmon/const"
	"github.com/b-harvest/Harvestmon/log"
	"github.com/b-harvest/Harvestmon/monitor/elmon/types"
	// "github.com/b-harvest/Harvestmon/repository"
	"github.com/b-harvest/Harvestmon/util"
	// "github.com/google/uuid"
	// "strconv"
	"time"
	logs "log"
)

func ExecutionMonitor(c *types.MonitorConfig, client *types.MonitorClient) {
	_, _, fn := util.TraceFirst()
	log.Debug("Starting monitor: " + fn)
	logs.Println("ExecutionMonitor started")
	logs.Printf("Running monitor: %+v with client: %+v\n", c, client)
	if client == nil {
        log.Error(errors.New("MonitorClient is nil in ExecutionMonitor"))
        return
    }

	createdAt := time.Now().UTC()

	blockNumber, err := client.GetBlockNumber()
	if err != nil {
		log.Error(errors.New(fmt.Sprintf("Error getting block number: %s", err)))
		return
	}
	log.Info(fmt.Sprintf("Current block number: %d", blockNumber))
	log.Info(fmt.Sprintf("Create At block number: %s", createdAt))

	// syncing, err := client.IsSyncing()
	// if err != nil {
	// 	log.Error(errors.New(fmt.Sprintf("Error checking if syncing: %s", err)))
	// 	return
	// }

	// if syncing {
	// 	log.Info("Node is syncing")
	// } else {
	// 	log.Info("Node is fully synced")
	// }
}