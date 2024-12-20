package repository

import (
	"gorm.io/gorm"
	"time"
)

type NodeFreeDiskStatus struct {
	Event Event `gorm:"foreignKey:EventUUID;references:EventUUID"`

	CreatedAt  time.Time `gorm:"primaryKey;column:created_at;not null;type:datetime(6);autoCreateTime:false"`
	EventUUID  string    `gorm:"primaryKey;column:event_uuid;not null;type:CHAR(36)"`
	Mountpoint string    `gorm:"primaryKey;column:mountpoint;not null;type:varchar(100)"`

	Device       string `gorm:"column:device;not null;type:varchar(100)"`
	FreeDiskSize uint64 `gorm:"column:free_disk_size;not null;type:bigint"`
}

func (*NodeFreeDiskStatus) TableName() string {
	return "node_free_disk_status"
}

func (s *NodeFreeDiskStatus) BeforeCreate(tx *gorm.DB) (err error) {
	return tx.Create(&s.Event).Error
}

type NodeCpuSecondsTotal struct {
	Event Event `gorm:"foreignKey:EventUUID;references:EventUUID"`

	CreatedAt time.Time `gorm:"primaryKey;column:created_at;not null;type:datetime(6);autoCreateTime:false"`
	EventUUID string    `gorm:"primaryKey;column:event_uuid;not null;type:CHAR(36)"`
	Mode      string    `gorm:"primaryKey;column:mode;not null;type:varchar(30)"`

	Number       int    `gorm:"column:number;not null;type:int"`
	SecondsTotal uint64 `gorm:"column:seconds_total;not null;type:bigint"`
}

func (*NodeCpuSecondsTotal) TableName() string {
	return "node_cpu_seconds_total"
}

func (s *NodeCpuSecondsTotal) BeforeCreate(tx *gorm.DB) (err error) {
	return tx.Create(&s.Event).Error
}

type NodeMemoryStatus struct {
	Event Event `gorm:"foreignKey:EventUUID;references:EventUUID"`

	CreatedAt time.Time `gorm:"primaryKey;column:created_at;not null;type:datetime(6);autoCreateTime:false"`
	EventUUID string    `gorm:"primaryKey;column:event_uuid;not null;type:CHAR(36)"`

	Total      uint64 `gorm:"column:total;not null;type:bigint"`
	Free       uint64 `gorm:"column:free;not null;type:bigint"`
	Buffer     uint64 `gorm:"column:buffer;not null;type:bigint"`
	Cached     uint64 `gorm:"column:cached;not null;type:bigint"`
	Slab       uint64 `gorm:"column:slab;not null;type:bigint"`
	PageTables uint64 `gorm:"column:page_tables;not null;type:bigint"`
	SwapCached uint64 `gorm:"column:swap_cached;not null;type:bigint"`
}

func (*NodeMemoryStatus) TableName() string {
	return "node_memory_status"
}

func (s *NodeMemoryStatus) BeforeCreate(tx *gorm.DB) (err error) {
	return tx.Create(&s.Event).Error
}

type NodeSystemdStatus struct {
	Event Event `gorm:"foreignKey:EventUUID;references:EventUUID"`

	CreatedAt   time.Time `gorm:"primaryKey;column:created_at;not null;type:datetime(6);autoCreateTime:false"`
	EventUUID   string    `gorm:"primaryKey;column:event_uuid;not null;type:CHAR(36)"`
	SystemdName string    `gorm:"primaryKey;column:name;not null;type:varchar(100)"`
	State       string    `gorm:"primaryKey;column:state;not null;type:varchar(50)"`
}

func (*NodeSystemdStatus) TableName() string {
	return "node_systemd_status"
}

func (s *NodeSystemdStatus) BeforeCreate(tx *gorm.DB) (err error) {
	return tx.Create(&s.Event).Error
}

type NodeNetworkTotal struct {
	Event Event `gorm:"foreignKey:EventUUID;references:EventUUID"`

	CreatedAt time.Time `gorm:"primaryKey;column:created_at;not null;type:datetime(6);autoCreateTime:false"`
	EventUUID string    `gorm:"primaryKey;column:event_uuid;not null;type:CHAR(36)"`
	Device    string    `gorm:"primaryKey;column:device;not null;type:varchar(100)"`

	ReceiveTotal  uint64 `gorm:"column:receive_total;not null;type:bigint"`
	TransmitTotal uint64 `gorm:"column:transmit_total;not null;type:bigint"`
}

func (*NodeNetworkTotal) TableName() string {
	return "node_network_total"
}

func (s *NodeNetworkTotal) BeforeCreate(tx *gorm.DB) (err error) {
	return tx.Create(&s.Event).Error
}
