package types

import (
	"database/sql"
	"fmt"
	"time"
)

// CheckerConfig specifies current version's commitId, database, etc...
type CheckerConfig struct {
	CommitId      string        `yaml:"commitId"`
	CheckInterval time.Duration `yaml:"checkInterval"`
	Database      Database      `yaml:"database"`
	HeightCheck   HeightCheck   `yaml:"heightCheck"`
	Heartbeat     Heartbeat     `yaml:"heartbeat"`
	PeerCheck     PeerCheck     `yaml:"peerCheck"`
	CommitCheck   CommitCheck   `yaml:"commitCheck"`
}

type HeightCheck struct {
	MaxStuckTime time.Duration `yaml:"maxStuckTime"`
}

type Heartbeat struct {
	MaxWaitTime time.Duration `yaml:"maxWaitTime"`
}

type PeerCheck struct {
	LowPeerCount int `yaml:"lowPeerCount"`
}

type CommitCheck struct {
	ValidatorAddress string `yaml:"validatorAddress"`
	MaxMissingCount  int    `yaml:"maxMissingCount"`
	TargetBlockCount int    `yaml:"targetBlockCount"`
}

// CheckerClient determines what alarmer should be used to send alarm associated with AlertLevelList.
type CheckerClient struct {
	DB *sql.DB
	// Key of AlertLevelList is same with AlertLevel.AlertName
	AlertLevelList map[string]AlertLevel
}

type AlertLevel struct {
	AlertName   string    `yaml:"alertName"`
	AlertLevel  string    `yaml:"alertLevel"`
	AlarmerList []Alarmer `yaml:"alarmerList"`
}

type Alarmer struct {
	AlarmerName   string     `yaml:"alarmerName"`
	Image         string     `yaml:"image"`
	AlamerENVList []AlamrEnv `yaml:"alamerENVList"`
}

type AlamrEnv struct {
	EnvName string `yaml:"envName"`
	Value   string `yaml:"value"`
}

type Database struct {
	User      string `yaml:"user"`
	Password  string `yaml:"password"`
	Host      string `yaml:"host"`
	Port      int    `yaml:"port"`
	DbName    string `yaml:"dbName"`
	AwsRegion string `yaml:"awsRegion"`
}

var (
	EnvTimeout      = "TIMEOUT"
	EnvAgentName    = "AGENT_NAME"
	EnvAgentHost    = "AGENT_HOST"
	EnvAgentPort    = "AGENT_PORT"
	EnvPushInterval = "PUSH_INTERVAL"
	EnvMonitors     = "AGENT_MONITORS"
	EnvCommitId     = "COMMIT_ID"
)

var (
	DefaultTimeout      = 3 * time.Second
	DefaultAgentName    = "instance"
	DefaultAgentHost    = "127.0.0.1"
	DefaultAgentPort    = 26657
	DefaultPushInterval = 10 * time.Second
)

// ApplyConfigFromEnvAndDefault will read the environmental variables into a config
// then validate it is reasonable and if there are not set in any column, set as defaults.
func (cfg *CheckerConfig) ApplyConfigFromEnvAndDefault() error {

	getDatabase(cfg)

	return nil
}

func parseEnvDuration(input string) (time.Duration, error) {
	duration, err := time.ParseDuration(input)
	if err != nil {
		return 0, fmt.Errorf("could not parse '%s' into a duration: %w", input, err)
	}

	if duration <= 0 {
		return 0, fmt.Errorf("must be greater than 0")
	}

	return duration, nil
}
