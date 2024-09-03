package types

import (
	"errors"
	"fmt"
	_const "github.com/b-harvest/Harvestmon/const"
	"github.com/b-harvest/Harvestmon/log"
	"github.com/b-harvest/Harvestmon/util"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var DEFAULT_AGENT_NAME AgentName = "defaultAgent"

// CheckerConfig specifies current version's commitId, database, etc...
type CheckerConfig struct {
	CommitId      string                      `yaml:"commitId"`
	CheckInterval *time.Duration              `yaml:"checkInterval"`
	AgentCheckers map[AgentName]*AgentChecker `yaml:"agentCheckers"`
}

type AgentChecker struct {
	HeightCheck *HeightCheck `yaml:"heightCheck"`
	// Heartbeat determine how long checker will wait for new event.
	// It could be specifiable by events name(etc: `tm:event:net_info`: 1m)
	Heartbeat   *map[string]*time.Duration `yaml:"heartbeat"`
	PeerCheck   *PeerCheck                 `yaml:"peerCheck"`
	CommitCheck *CommitCheck               `yaml:"commitCheck"`
}

const DefaultMaxWaitTimeKey = "maxWaitTime"

type HeightCheck struct {
	MaxStuckTime *time.Duration `yaml:"maxStuckTime"`
}

type PeerCheck struct {
	LowPeerCount int `yaml:"lowPeerCount"`
}

type CommitCheck struct {
	ValidatorAddress string `yaml:"validatorAddress"`
	MaxMissingCount  int    `yaml:"maxMissingCount"`
	TargetBlockCount int    `yaml:"targetBlockCount"`
}

var (
	EnvCommitId                  = "COMMIT_ID"
	EnvCheckInterval             = "CHECK_INTERVAL"
	EnvHeightMaxStuckTime        = "HEIGHT_MAX_STUCK_TIME"
	EnvHeartbeatMaxWaitTime      = "HEARTBEAT_MAX_WAIT_TIME"
	EnvLowPeerCount              = "LOW_PEER_COUNT"
	EnvCommitCheckValAddr        = "COMMIT_CHECK_VALIDATOR_ADDRESS"
	EnvCommitCheckMaxMissingCnt  = "COMMIT_CHECK_MAX_MISSING_COUNT"
	EnvCommitCheckTargetBlockCnt = "COMMIT_CHECK_TARGET_BLOCK_COUNT"

	EnvGithubOwner  = "GITHUB_OWNER"
	EnvGithubRepo   = "GITHUB_REPO"
	EnvGithubBranch = "GITHUB_BRANCH"
	EnvGithubToken  = "GITHUB_TOKEN"

	EnvGithubServiceAlertFile = "GITHUB_SERVICE_ALERT_FILE"
	EnvGithubCustomAgentFiles = "GITHUB_CUSTOM_AGENT_FILES"

	EnvCheckerFunction = "CHECKER"
)

var (
	DefaultCheckInterval             = 10 * time.Second
	DefaultHeightMaxStuckTime        = 5 * time.Minute
	DefaultHeartbeatMaxWaitTime      = 3 * time.Minute
	DefaultLowPeerCount              = 5
	DefaultCommitCheckMaxMissingCnt  = 10
	DefaultCommitCheckTargetBlockCnt = 50
)

// ApplyConfigFromEnvAndDefault will read the environmental variables into a config
// then validate it is reasonable and if there are not set in any column, set as defaults.
func (cfg *CheckerConfig) ApplyConfigFromEnvAndDefault() error {

	if cfg.CheckInterval == nil {
		v := os.Getenv(EnvCheckInterval)
		if v == "" {
			cfg.CheckInterval = &DefaultCheckInterval
			log.Debug("CheckInterval set as default: " + cfg.CheckInterval.String())
		} else {
			checkInterval, err := parseEnvDuration(v)
			if err != nil {
				return errors.New(err.Error())
			}
			cfg.CheckInterval = &checkInterval
			log.Debug("CheckInterval set as ENV: " + cfg.CheckInterval.String())
		}
	} else {
		log.Debug("CheckInterval set as " + cfg.CheckInterval.String())
	}

	if cfg.AgentCheckers[DEFAULT_AGENT_NAME] == nil {
		ac := AgentChecker{}
		cfg.AgentCheckers[DEFAULT_AGENT_NAME] = &ac
	}

	if cfg.AgentCheckers[DEFAULT_AGENT_NAME].HeightCheck == nil || cfg.AgentCheckers[DEFAULT_AGENT_NAME].HeightCheck.MaxStuckTime == nil {
		v := os.Getenv(EnvHeightMaxStuckTime)
		if v == "" {
			cfg.AgentCheckers[DEFAULT_AGENT_NAME].HeightCheck = &HeightCheck{MaxStuckTime: &DefaultHeightMaxStuckTime}
			log.Debug("HeightMaxStuckTime set as default: " + cfg.AgentCheckers[DEFAULT_AGENT_NAME].HeightCheck.MaxStuckTime.String())
		} else {
			maxStuckTime, err := parseEnvDuration(v)
			if err != nil {
				return errors.New(err.Error())
			}
			cfg.AgentCheckers[DEFAULT_AGENT_NAME].HeightCheck = &HeightCheck{MaxStuckTime: &maxStuckTime}
			log.Debug("HeightMaxStuckTime set as ENV: " + cfg.AgentCheckers[DEFAULT_AGENT_NAME].HeightCheck.MaxStuckTime.String())
		}
	} else {
		log.Debug("HeightMaxStuckTime set as " + cfg.CheckInterval.String())
	}

	if _, exists := (*cfg.AgentCheckers[DEFAULT_AGENT_NAME].Heartbeat)[DefaultMaxWaitTimeKey]; !exists {
		v := os.Getenv(EnvHeartbeatMaxWaitTime)
		if v == "" {
			(*cfg.AgentCheckers[DEFAULT_AGENT_NAME].Heartbeat)[DefaultMaxWaitTimeKey] = &DefaultHeartbeatMaxWaitTime
			log.Debug("HeartbeatMaxWaitTime set as default: " + (*cfg.AgentCheckers[DEFAULT_AGENT_NAME].Heartbeat)[DefaultMaxWaitTimeKey].String())
		} else {
			maxWaitTime, err := parseEnvDuration(v)
			if err != nil {
				return errors.New(err.Error())
			}
			(*cfg.AgentCheckers[DEFAULT_AGENT_NAME].Heartbeat)[DefaultMaxWaitTimeKey] = &maxWaitTime
			log.Debug("HeartbeatMaxWaitTime set as ENV: " + (*cfg.AgentCheckers[DEFAULT_AGENT_NAME].Heartbeat)[DefaultMaxWaitTimeKey].String())
		}
	} else {
		log.Debug("HeartbeatMaxWaitTime set as " + (*cfg.AgentCheckers[DEFAULT_AGENT_NAME].Heartbeat)[DefaultMaxWaitTimeKey].String())
	}

	if cfg.AgentCheckers[DEFAULT_AGENT_NAME].PeerCheck == nil || cfg.AgentCheckers[DEFAULT_AGENT_NAME].PeerCheck.LowPeerCount == 0 {
		v := os.Getenv(EnvLowPeerCount)
		if v == "" {
			cfg.AgentCheckers[DEFAULT_AGENT_NAME].PeerCheck = &PeerCheck{LowPeerCount: DefaultLowPeerCount}
			log.Debug("LowPeerCount set as default: " + strconv.Itoa(cfg.AgentCheckers[DEFAULT_AGENT_NAME].PeerCheck.LowPeerCount))
		} else {
			lowPeerCount, err := strconv.Atoi(v)
			if err != nil {
				return errors.New(err.Error())
			}
			cfg.AgentCheckers[DEFAULT_AGENT_NAME].PeerCheck = &PeerCheck{LowPeerCount: lowPeerCount}
			log.Debug("LowPeerCount set as ENV: " + strconv.Itoa(cfg.AgentCheckers[DEFAULT_AGENT_NAME].PeerCheck.LowPeerCount))
		}
	} else {
		log.Debug("LowPeerCount set as " + strconv.Itoa(cfg.AgentCheckers[DEFAULT_AGENT_NAME].PeerCheck.LowPeerCount))
	}

	if cfg.AgentCheckers[DEFAULT_AGENT_NAME].CommitCheck == nil || (cfg.AgentCheckers[DEFAULT_AGENT_NAME].CommitCheck.ValidatorAddress == "" && os.Getenv(EnvCommitCheckValAddr) == "") {
		cfg.AgentCheckers[DEFAULT_AGENT_NAME].CommitCheck = &CommitCheck{}
		log.Debug("BlockCommit check feature will be disabled.")
	} else {
		if cfg.AgentCheckers[DEFAULT_AGENT_NAME].CommitCheck.ValidatorAddress == "" {
			v := os.Getenv(EnvCommitCheckValAddr)
			cfg.AgentCheckers[DEFAULT_AGENT_NAME].CommitCheck.ValidatorAddress = v
			log.Debug("CommitMaxMissingCount set as ENV: " + cfg.AgentCheckers[DEFAULT_AGENT_NAME].CommitCheck.ValidatorAddress)
		}

		if cfg.AgentCheckers[DEFAULT_AGENT_NAME].CommitCheck.MaxMissingCount == 0 {
			v := os.Getenv(EnvCommitCheckMaxMissingCnt)
			if v == "" {
				cfg.AgentCheckers[DEFAULT_AGENT_NAME].CommitCheck.MaxMissingCount = DefaultCommitCheckMaxMissingCnt
				log.Debug("CommitMaxMissingCount set as default: " + strconv.Itoa(cfg.AgentCheckers[DEFAULT_AGENT_NAME].CommitCheck.MaxMissingCount))
			} else {
				maxMissingCount, err := strconv.Atoi(v)
				if err != nil {
					return errors.New(err.Error())
				}
				cfg.AgentCheckers[DEFAULT_AGENT_NAME].CommitCheck.MaxMissingCount = maxMissingCount
				log.Debug("CommitMaxMissingCount set as ENV: " + strconv.Itoa(cfg.AgentCheckers[DEFAULT_AGENT_NAME].CommitCheck.MaxMissingCount))
			}
		} else {
			log.Debug("CommitMaxMissingCount set as " + strconv.Itoa(cfg.AgentCheckers[DEFAULT_AGENT_NAME].CommitCheck.MaxMissingCount))
		}

		if cfg.AgentCheckers[DEFAULT_AGENT_NAME].CommitCheck.TargetBlockCount == 0 {
			v := os.Getenv(EnvCommitCheckTargetBlockCnt)
			if v == "" {
				cfg.AgentCheckers[DEFAULT_AGENT_NAME].CommitCheck.TargetBlockCount = DefaultCommitCheckTargetBlockCnt
				log.Debug("TargetBlockCount set as default: " + strconv.Itoa(cfg.AgentCheckers[DEFAULT_AGENT_NAME].CommitCheck.TargetBlockCount))
			} else {
				targetBlockCount, err := strconv.Atoi(v)
				if err != nil {
					return errors.New(err.Error())
				}
				cfg.AgentCheckers[DEFAULT_AGENT_NAME].CommitCheck.TargetBlockCount = targetBlockCount
				log.Debug("TargetBlockCount set as ENV: " + strconv.Itoa(cfg.AgentCheckers[DEFAULT_AGENT_NAME].CommitCheck.TargetBlockCount))
			}
		} else {
			log.Debug("TargetBlockCount set as " + strconv.Itoa(cfg.AgentCheckers[DEFAULT_AGENT_NAME].CommitCheck.TargetBlockCount))
		}
	}

	if cfg.CommitId == "" {
		v := os.Getenv(EnvCommitId)
		if v == "" {
			return errors.New("no commit id found. please set commit id through default_checker_rules.yaml or env($COMMIT_ID)")
		}
		cfg.CommitId = v
		log.Debug("CommitID set as ENV: " + cfg.CommitId)
	} else {
		log.Debug("CommitID set as " + cfg.CommitId)
	}

	return nil
}

func GetCustomAgentFiles() []CustomAgentConfig {
	var (
		githubOwner = os.Getenv(EnvGithubOwner)
		repo        = os.Getenv(EnvGithubRepo)
		branch      = os.Getenv(EnvGithubBranch)
		githubToken = os.Getenv(EnvGithubToken)

		githubServiceCustomAgentFiles = os.Getenv(EnvGithubCustomAgentFiles)

		githubFiles []string
		err         error

		agentConfigs []CustomAgentConfig
	)

	if repo != "" {
		for _, customAgentFilePath := range strings.Split(githubServiceCustomAgentFiles, ",") {

			githubFiles, err = util.FetchGithubFile(githubOwner, repo, branch, customAgentFilePath, githubToken)

			for _, githubFile := range githubFiles {
				var agentConfig CustomAgentConfig

				err = yaml.Unmarshal([]byte(githubFile), &agentConfig)
				if err != nil {
					log.Warn(err.Error())
					continue
				}

				agentConfigs = append(agentConfigs, agentConfig)
			}
		}
	}
	return agentConfigs
}

func (c *CheckerConfig) MergeWithCustomAgentChecker(agentConfigs []CustomAgentConfig) {

	for _, agentConfig := range agentConfigs {
		if c.AgentCheckers[agentConfig.AgentName] == nil {
			c.AgentCheckers[agentConfig.AgentName] = new(AgentChecker)
		}
		if agentConfig.AgentChecker != nil {
			c.AgentCheckers[agentConfig.AgentName] = agentConfig.AgentChecker

			if agentConfig.AgentChecker.CommitCheck == nil {
				c.AgentCheckers[agentConfig.AgentName].CommitCheck = c.AgentCheckers[DEFAULT_AGENT_NAME].CommitCheck
			}
			if agentConfig.AgentChecker.Heartbeat == nil {
				c.AgentCheckers[agentConfig.AgentName].Heartbeat = c.AgentCheckers[DEFAULT_AGENT_NAME].Heartbeat
			}
			if agentConfig.AgentChecker.HeightCheck == nil {
				c.AgentCheckers[agentConfig.AgentName].HeightCheck = c.AgentCheckers[DEFAULT_AGENT_NAME].HeightCheck
			}
			if agentConfig.AgentChecker.PeerCheck == nil {
				c.AgentCheckers[agentConfig.AgentName].PeerCheck = c.AgentCheckers[DEFAULT_AGENT_NAME].PeerCheck
			}
		} else {
			c.AgentCheckers[agentConfig.AgentName] = c.AgentCheckers[DEFAULT_AGENT_NAME]
		}
	}

	delete(c.AgentCheckers, DEFAULT_AGENT_NAME)
}

type Alert struct {
	Alarmer    Alarmer
	Message    string
	AlertLevel AlertLevel
	Agent      AgentName
}

func NewAlert(alarmer Alarmer, alertLevel AlertLevel, agentName AgentName, msg string) Alert {
	var content string
	if alarmer.Format == HTML_ALARM_MESSAGE_FORMAT {
		content = aHtmlprintf(agentName, alertLevel, _const.HARVESTMON_TENDERMINT_SERVICE_NAME, msg)
	} else if alarmer.Format == CUSTOM_ALARM_MESSAGE_FORMAT {
		content = msg
	} else {
		content = aPlainprintf(agentName, alertLevel, _const.HARVESTMON_TENDERMINT_SERVICE_NAME, msg)
	}

	return Alert{
		Alarmer:    alarmer,
		Message:    content,
		AlertLevel: alertLevel,
		Agent:      agentName,
	}
}

func aHtmlprintf(agent AgentName, alertLevel AlertLevel, service, msg string) string {
	alertNames := strings.Split(alertLevel.AlertName.String(), ":")

	return fmt.Sprintf("<b>%s</b>\n\n"+
		"AlertName: %s \n"+
		"AlertLevel: <b>%s</b> \n"+
		"Service: %s\n\n%s", agent, alertNames[len(alertNames)-1], alertLevel.AlertLevel, service, msg)

}

func aPlainprintf(agent AgentName, alertLevel AlertLevel, service, msg string) string {
	alertNames := strings.Split(alertLevel.AlertName.String(), ":")

	return fmt.Sprintf("%s\n\n"+
		"AlertName: %s \n"+
		"AlertLevel: %s \n"+
		"Service: %s\n\n%s", agent, alertNames[len(alertNames)-1], alertLevel.AlertLevel, service, msg)

}

type AlertName string

func (a *AlertName) String() string {
	return string(*a)
}

func (alertLevelName *AlertName) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var levelName string
	if err := unmarshal(&levelName); err != nil {
		return err
	}
	*alertLevelName = AlertName(levelName)
	return nil
}

type AlertLevel struct {
	AlertName  AlertName `yaml:"name"`
	AlertLevel string    `yaml:"level"`
}

type Alarmer struct {
	TargetLevels        []string             `yaml:"targetLevels"`
	AlarmerName         string               `yaml:"name"`
	AlarmParamList      map[string]any       `yaml:"params"`
	Format              AlarmerMessageFormat `yaml:"format"`
	AlarmResendDuration *time.Duration       `yaml:"alarmResendDuration"`
}

type AlarmerMessageFormat string

var (
	CUSTOM_ALARM_MESSAGE_FORMAT AlarmerMessageFormat = "custom"
	HTML_ALARM_MESSAGE_FORMAT   AlarmerMessageFormat = "html"
	PLAIN_ALARM_MESSAGE_FORMAT  AlarmerMessageFormat = "plain"

	AVAILABLE_FORMAT_LIST = []AlarmerMessageFormat{
		CUSTOM_ALARM_MESSAGE_FORMAT,
		HTML_ALARM_MESSAGE_FORMAT,
		PLAIN_ALARM_MESSAGE_FORMAT,
	}
)

func (alarmerMessageFormat *AlarmerMessageFormat) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var format string
	if err := unmarshal(&format); err != nil {
		return err
	}

	for _, availableFormat := range AVAILABLE_FORMAT_LIST {
		if string(availableFormat) == format {
			*alarmerMessageFormat = AlarmerMessageFormat(format)
			return nil
		}
	}

	return errors.New(fmt.Sprintf("not acceptable value. you should enter defined format (%v)", AVAILABLE_FORMAT_LIST))
}

type CustomAgentConfig struct {
	AgentName    AgentName     `yaml:"agentName"`
	AgentChecker *AgentChecker `yaml:"checker"`
	AlertLevel   []AlertLevel  `yaml:"alert"`
	Alarmer      []Alarmer     `yaml:"alarmer"`
}

func (agentName *AgentName) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var levelName string
	if err := unmarshal(&levelName); err != nil {
		return err
	}
	*agentName = AgentName(levelName)
	return nil
}

type AlertDefinition struct {
	AlertLevel []AlertLevel `yaml:"alert"`
	Alarmer    []Alarmer    `yaml:"alarmer"`
}

func ParseAlertDefinition() (*AlertDefinition, error) {
	var (
		defaultAlertBytes []byte
		defaultAlert      = AlertDefinition{
			Alarmer:    []Alarmer{},
			AlertLevel: []AlertLevel{},
		}
	)

	pwd, err := os.Getwd()
	defaultAlertBytes, err = os.ReadFile(filepath.Join(pwd, "resources/default_alert_definition.yaml"))
	if err != nil {
		log.Fatal(err)
	}

	err = yaml.Unmarshal(defaultAlertBytes, &defaultAlert)
	if err != nil {
		log.Fatal(err)
	}

	var (
		githubOwner = os.Getenv(EnvGithubOwner)
		repo        = os.Getenv(EnvGithubRepo)
		branch      = os.Getenv(EnvGithubBranch)
		githubToken = os.Getenv(EnvGithubToken)

		githubServiceAlertFile = os.Getenv(EnvGithubServiceAlertFile)

		githubFiles []string
	)
	if repo != "" {
		githubFiles, err = util.FetchGithubFile(githubOwner, repo, branch, githubServiceAlertFile, githubToken)

		for _, githubFile := range githubFiles {
			var customAlertDefinition AlertDefinition

			err = yaml.Unmarshal([]byte(githubFile), &customAlertDefinition)
			if err != nil {
				return nil, err
			}

			for _, v := range customAlertDefinition.AlertLevel {
				var exists bool
				for idx, x := range defaultAlert.AlertLevel {
					if x.AlertName == v.AlertName {
						exists = true
						defaultAlert.AlertLevel[idx].AlertLevel = v.AlertLevel
					}
				}
				if !exists {
					defaultAlert.AlertLevel = append(defaultAlert.AlertLevel, v)
				}
			}
			for _, v := range customAlertDefinition.Alarmer {
				defaultAlert.Alarmer = append(defaultAlert.Alarmer, v)
			}
		}
	}

	return &defaultAlert, nil
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

func ParseCheckerFunctions(defaultCheckerRegistry map[string]Func) []Checker {
	var result []Checker
	checkerFunctions := strings.Split(os.Getenv(EnvCheckerFunction), ",")
	if checkerFunctions != nil {
		for _, checkerName := range checkerFunctions {
			if checkerFunction, exists := defaultCheckerRegistry[checkerName]; exists {
				result = append(result, checkerFunction)
			}
		}
	}

	if len(result) == 0 {
		for _, checkerFunction := range defaultCheckerRegistry {
			result = append(result, checkerFunction)
		}
	}

	return result
}
