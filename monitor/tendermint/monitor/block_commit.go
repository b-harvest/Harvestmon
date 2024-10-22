package monitor

import (
	"errors"
	"fmt"
	_const "github.com/b-harvest/Harvestmon/const"
	"github.com/b-harvest/Harvestmon/log"
	"github.com/b-harvest/Harvestmon/monitor/tendermint/types"
	"github.com/b-harvest/Harvestmon/repository"
	"github.com/b-harvest/Harvestmon/util"
	"github.com/google/uuid"
	"strconv"
	"sync"
	"time"
)

func BlockCommitMonitor(c *types.MonitorConfig, client *types.MonitorClient) {
	_, _, fn := util.TraceFirst()
	log.Debug("Starting monitor: " + fn)

	commitMonitorRepository := repository.CommitRepository{BaseRepository: repository.BaseRepository{DB: *client.GetDatabase(c.DbBatchSize)}}

	status, err := client.GetCometBFTStatus()
	if err != nil {
		log.Error(err)
	}
	if status == nil {
		log.Error(errors.New("cometBFT_status monitor response <nil>"))
		return
	}

	latestHeight, err := strconv.ParseUint(status.SyncInfo.LatestBlockHeight, 0, 64)
	if err != nil {
		log.Error(err)
	}

	metaRepository := repository.MetaMonitorRepository{BaseRepository: repository.BaseRepository{DB: *client.GetDatabase(c.DbBatchSize)}}
	startHeight, err := metaRepository.FetchHighestHeight(c.Agent.AgentName)
	if err != nil {
		log.Debug(err.Error())
		startHeight = latestHeight - 1
	}

	if (latestHeight - startHeight) > (uint64(c.Agent.PushInterval.Seconds()) * 200) {
		startHeight = latestHeight - (uint64(c.Agent.PushInterval.Seconds()) * 200)
		log.Info(fmt.Sprintf("[block_commit] distance from startHeight to latestHeight is too large. automatically set startHeight as %d", startHeight))
	}

	for i := startHeight; i < latestHeight; i += uint64(c.GetBatchSize()) {
		var (
			wg         sync.WaitGroup
			recordChan = make(chan repository.TendermintCommit, c.GetBatchSize())
		)

		semaphore := make(chan struct{}, c.Agent.BlockCommitMaxConcurrency)

		// Adjust the upper bound to avoid fetching heights beyond latestHeight
		batchEnd := i + uint64(c.GetBatchSize())
		if batchEnd > latestHeight {
			batchEnd = latestHeight
		}

		for j := i; j < batchEnd; j++ {
			wg.Add(1)
			go processHeight(j, client, recordChan, c, &wg, semaphore)
		}

		go func() {
			wg.Wait()
			close(recordChan)
		}()

		var tcRecords []repository.TendermintCommit
		for record := range recordChan {
			tcRecords = append(tcRecords, record)
		}

		err = commitMonitorRepository.CreateBatch(tcRecords)
		if err != nil {
			log.Error(err)
		}
	}

	err = metaRepository.Save(repository.MetaMonitor{AgentName: c.Agent.AgentName, Height: int64(latestHeight)})
	if err != nil {
		log.Error(err)
	}

	log.Debug("Complete monitor: " + fn)
}

func processHeight(i uint64, client *types.MonitorClient, recordChan chan repository.TendermintCommit, c *types.MonitorConfig, wg *sync.WaitGroup, semaphore chan struct{}) {
	defer wg.Done()
	semaphore <- struct{}{}        // Acquire a spot in the semaphore
	defer func() { <-semaphore }() // Release the spot in the semaphore when done

	commit, err := client.GetCommitWithHeight(i)
	if err != nil {
		log.Error(errors.New(fmt.Sprintf("Error fetching commit: %v", err)))
		return
	}

	eventUUID, err := uuid.NewUUID()
	if err != nil {
		log.Error(errors.New(fmt.Sprintf("Error generating UUID: %v", err)))
		return
	}

	createdAt := time.Now().UTC()

	var signatures []repository.TendermintCommitSignature

	var commitSigs = commit.Result.SignedHeader.Commit.Signatures
	if len(commitSigs) == 0 && len(commit.Result.SignedHeader.Commit.Precommits) > 0 {
		commitSigs = commit.Result.SignedHeader.Commit.Precommits
	}

	for _, signature := range commitSigs {
		if signature.ValidatorAddress == "" {
			continue
		}
		signatures = append(signatures, repository.TendermintCommitSignature{
			ValidatorAddress:          signature.ValidatorAddress,
			TendermintCommitCreatedAt: createdAt,
			EventUUID:                 eventUUID.String(),
			Timestamp:                 signature.Timestamp,
			Signature:                 signature.Signature,
			BlockIdFlag:               signature.BlockIDFlag,
		})
	}

	result := repository.TendermintCommit{
		CreatedAt: createdAt,
		EventUUID: eventUUID.String(),
		Event: repository.Event{
			EventUUID:   eventUUID.String(),
			AgentName:   c.Agent.AgentName,
			ServiceName: _const.HARVESTMON_TENDERMINT_SERVICE_NAME,
			CommitID:    c.Agent.CommitId,
			EventType:   _const.TM_COMMIT_EVENT_TYPE,
			CreatedAt:   createdAt,
		},
		ChainID:            commit.Result.ChainID,
		Height:             commit.Result.Height,
		Time:               commit.Result.Time,
		LastBlockIdHash:    commit.Result.LastBlockID.Hash,
		LastCommitHash:     commit.Result.LastCommitHash,
		DataHash:           commit.Result.DataHash,
		ValidatorsHash:     commit.Result.ValidatorsHash,
		NextValidatorsHash: commit.Result.NextValidatorsHash,
		ConsensusHash:      commit.Result.ConsensusHash,
		AppHash:            commit.Result.AppHash,
		LastResultsHash:    commit.Result.LastResultsHash,
		EvidenceHash:       commit.Result.EvidenceHash,
		ProposerAddress:    commit.Result.ProposerAddress,
		Round:              commit.Result.Commit.Round,
		CommitBlockIdHash:  commit.Result.Commit.BlockID.Hash,
		Signatures:         signatures,
	}

	recordChan <- result

	log.Info(fmt.Sprintf("[block_commit] height: %v, signature count: %d", i, len(signatures)))
	if err != nil {
		log.Error(errors.New(fmt.Sprintf("Error saving commit: %v", err)))
	}
}
