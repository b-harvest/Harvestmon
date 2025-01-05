package repository

import (
	"errors"
	"gorm.io/gorm"
)

type MonitorRepository interface {
	Save(any ...any) error
}

type Repository struct {
	DB gorm.DB
}

type StoreEntity interface{}

func (r *Repository) Save(event interface{}) error {
	res := r.DB.Save(event)
	if res.Error != nil {
		return res.Error
	}
	return nil
}

func (r *Repository) SaveAll(events []StoreEntity) error {
	if len(events) == 0 {
		return nil
	}

	return r.DB.Transaction(func(tx *gorm.DB) error {
		for _, e := range events {
			if e == nil {
				return errors.New("event is nil")
			}

			res := tx.Create(e)
			if res.Error != nil {
				return res.Error
			}
		}
		return nil
	})
}
