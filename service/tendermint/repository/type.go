package repository

import "time"

type Event struct {
	EventUUID   string
	AgentName   string
	ServiceName string
	CommitID    string
	EventType   string
	CreatedAt   time.Time
}

type MonitorRepository interface {
	Save(any ...any) error
}
