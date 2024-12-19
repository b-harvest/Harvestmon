package main

import (
	"github.com/b-harvest/Harvestmon/repository"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

type Store struct {
	logger *log.Logger

	Interval *time.Duration `mapstructure:"interval"`

	poolingSize int

	repo *repository.BaseRepository

	waitingMtx      sync.Mutex
	waitingEntities []StoreEntity
}

func (s *Store) Validate() error {
	if s.repo == nil {
		return errors.New("repo is required")
	}
	if s.Interval == nil {
		return errors.New("timeout is required")
	}

	return nil
}

func (s *Store) StartWithStoreQueue(storeQueue <-chan StoreEntity) {
	s.logger.Infof("starting store. interval: %v", s.Interval)

	timeout := time.NewTicker(*s.Interval)

	var (
		shouldStore bool
	)
	for {
		select {
		case entity := <-storeQueue:
			s.waitingMtx.Lock()
			s.waitingEntities = append(s.waitingEntities, entity)
			s.waitingMtx.Unlock()
			if len(s.waitingEntities) < s.poolingSize {
				// skipping to store...
				continue
			} else {
				s.logger.Print("store queue is full. saving...")
				shouldStore = true
			}
		case <-timeout.C:
			s.logger.Printf("store timeout(%v) over. saving...", s.Interval)
			shouldStore = true
		}

		if len(s.waitingEntities) > 0 && shouldStore {

			// store entities
			errs := s.store()
			for _, e := range errs {
				s.logger.WithError(e).Error("storing error")
			}

			// cleanup
			s.waitingEntities = nil
			shouldStore = false
		}
	}
}

func (s *Store) store() []error {
	var (
		errs []error
		err  error

		tmCommits       []repository.TendermintCommit
		tmStatuses      []repository.TendermintStatus
		tmNetInfos      []repository.TendermintNetInfo
		ethBlockNumbers []repository.EthereumBlockNumber
	)
	s.waitingMtx.Lock()
	for _, entity := range s.waitingEntities {
		if entity.tmCommit != nil {
			tmCommits = append(tmCommits, *entity.tmCommit)
		} else if entity.tmStatus != nil {
			tmStatuses = append(tmStatuses, *entity.tmStatus)
		} else if entity.tmNetInfo != nil {
			tmNetInfos = append(tmNetInfos, *entity.tmNetInfo)
		} else if entity.ethBlock != nil {
			ethBlockNumbers = append(ethBlockNumbers, *entity.ethBlock)
		}
	}
	s.waitingMtx.Unlock()

	tmCommitRepository := repository.TendermintCommitRepository{BaseRepository: *s.repo}
	err = tmCommitRepository.SaveAll(tmCommits)
	if err != nil {
		errs = append(errs, err)
	}

	tmStoreRepository := repository.TendermintStatusRepository{BaseRepository: *s.repo}
	err = tmStoreRepository.SaveAll(tmStatuses)
	if err != nil {
		errs = append(errs, err)
	}

	tmNetInfoRepository := repository.TendermintNetInfoRepository{BaseRepository: *s.repo}
	err = tmNetInfoRepository.SaveAll(tmNetInfos)
	if err != nil {
		errs = append(errs, err)
	}

	ethBlockNumberRepository := repository.EthBlockNumberRepository{BaseRepository: *s.repo}
	err = ethBlockNumberRepository.SaveAll(ethBlockNumbers)
	if err != nil {
		errs = append(errs, err)
	}

	return errs
}
