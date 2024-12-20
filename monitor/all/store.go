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

	repo *repository.Repository

	waitingMtx      sync.Mutex
	waitingEntities []repository.StoreEntity
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

func (s *Store) StartWithStoreQueue(storeQueue <-chan repository.StoreEntity) {
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
		errs            []error
		err             error
		storingEntities []repository.StoreEntity
	)
	s.waitingMtx.Lock()
	for _, entity := range s.waitingEntities {
		if entity != nil {
			storingEntities = append(storingEntities, entity)
		}
	}
	s.waitingMtx.Unlock()

	err = s.repo.SaveAll(storingEntities)
	if err != nil {
		errs = append(errs, err)
	}

	return errs
}
