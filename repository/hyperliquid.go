package repository

import (
	"gorm.io/gorm"
	"time"
)

type HyperliquidStatus struct {
	CreatedAt time.Time `gorm:"primaryKey;column:created_at;not null;type:datetime(6);autoCreateTime:false"`
	Event     Event     `gorm:"foreignKey:EventUUID;references:EventUUID"`
	EventUUID string    `gorm:"primaryKey;column:event_uuid;not null;type:CHAR(36)"`

	Timestamp     time.Time `gorm:"column:timestamp;not null;type:datetime(6);autoCreateTime:false"`
	Round         uint64
	HomeValidator string                       `gorm:"column:home_validator;not null;type:varchar(100)"`
	Validators    []HyperliquidStatusValidator `gorm:"foreignKey:TendermintCommitCreatedAt,EventUUID;references:CreatedAt,EventUUID"`
}

type HyperliquidStatusValidator struct {
	HyperliquidStatus          HyperliquidStatus `gorm:"foreignKey:HyperliquidStatusCreatedAt,EventUUID;references:CreatedAt,EventUUID"`
	HyperliquidStatusCreatedAt time.Time         `gorm:"primaryKey;column:hyperliquid_status_created_at;not null;type:datetime(6)"`
	Event                      Event             `gorm:"foreignKey:EventUUID;references:EventUUID"`
	EventUUID                  string            `gorm:"primaryKey;column:event_uuid;not null;type:CHAR(36)"`

	ValidatorAddress string `gorm:"primaryKey;column:validator_address;not null;type:varchar(100)"`

	Stakes             uint64 `gorm:"column:stakes;not null;type:bigint"`
	IsJailed           bool   `gorm:"column:is_jailed;not null;type:boolean"`
	IsNext             bool   `gorm:"column:is_next;not null;type:boolean"`
	IsMissingHeartbeat bool   `gorm:"column:is_missing_heartbeat;not null;type:boolean"`

	SinceLastSuccess float64  `gorm:"column:since_last_success;not null;type:float"`
	LastAckDuration  *float64 `gorm:"column:last_ack_duration;not null;type:float"`
}

func (*HyperliquidStatus) TableName() string {
	return "hyperliquid_status"
}

func (s *HyperliquidStatus) BeforeCreate(tx *gorm.DB) (err error) {
	for _, validator := range s.Validators {
		err = tx.Create(&validator).Error
	}

	err = tx.Create(&s.Event).Error
	if err != nil {
		return err
	}
	return
}

func (s *HyperliquidStatusValidator) TableName() string {
	return "hyperliquid_status_validator"
}
