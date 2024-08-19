package types

import (
	"context"
	"errors"
	"fmt"
	_const "github.com/b-harvest/Harvestmon/const"
	database "github.com/b-harvest/Harvestmon/database"
	"github.com/b-harvest/Harvestmon/log"
	"github.com/b-harvest/Harvestmon/util"
	"gopkg.in/yaml.v3"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var DEFAULT_AGENT_NAME AgentName = "defaultAgent"

// CheckerConfig specifies current version's commitId, database, etc...
type CheckerConfig struct {
	CommitId      string                      `yaml:"commitId"`
	CheckInterval *time.Duration              `yaml:"checkInterval"`
	Database      database.Database           `yaml:"database"`
	AgentCheckers map[AgentName]*AgentChecker `yaml:"agentCheckers"`
}

type AgentChecker struct {
	HeightCheck *HeightCheck `yaml:"heightCheck"`
	Heartbeat   *Heartbeat   `yaml:"heartbeat"`
	PeerCheck   *PeerCheck   `yaml:"peerCheck"`
	CommitCheck *CommitCheck `yaml:"commitCheck"`
}

type HeightCheck struct {
	MaxStuckTime *time.Duration `yaml:"maxStuckTime"`
}

type Heartbeat struct {
	MaxWaitTime *time.Duration `yaml:"maxWaitTime"`
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

	EnvAlertDefinitionPlace = "ALERT_DEFINITION"

	EnvGithubOwner  = "GITHUB_OWNER"
	EnvGithubRepo   = "GITHUB_REPO"
	EnvGithubBranch = "GITHUB_BRANCH"
	EnvGithubPath   = "GITHUB_PATH"
	EnvGithubToken  = "GITHUB_TOKEN"

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

	if cfg.AgentCheckers[DEFAULT_AGENT_NAME].Heartbeat == nil || cfg.AgentCheckers[DEFAULT_AGENT_NAME].Heartbeat.MaxWaitTime == nil {
		v := os.Getenv(EnvHeartbeatMaxWaitTime)
		if v == "" {
			cfg.AgentCheckers[DEFAULT_AGENT_NAME].Heartbeat = &Heartbeat{MaxWaitTime: &DefaultHeartbeatMaxWaitTime}
			log.Debug("HeartbeatMaxWaitTime set as default: " + cfg.AgentCheckers[DEFAULT_AGENT_NAME].Heartbeat.MaxWaitTime.String())
		} else {
			maxWaitTime, err := parseEnvDuration(v)
			if err != nil {
				return errors.New(err.Error())
			}
			cfg.AgentCheckers[DEFAULT_AGENT_NAME].Heartbeat = &Heartbeat{MaxWaitTime: &maxWaitTime}
			log.Debug("HeartbeatMaxWaitTime set as ENV: " + cfg.AgentCheckers[DEFAULT_AGENT_NAME].Heartbeat.MaxWaitTime.String())
		}
	} else {
		log.Debug("HeartbeatMaxWaitTime set as " + cfg.AgentCheckers[DEFAULT_AGENT_NAME].Heartbeat.MaxWaitTime.String())
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
			return errors.New("no commit id found. please set commit id through config.yaml or env($COMMIT_ID)")
		}
		cfg.CommitId = v
		log.Debug("CommitID set as ENV: " + cfg.CommitId)
	} else {
		log.Debug("CommitID set as " + cfg.CommitId)
	}

	return nil
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
	MKDOWN_ALARM_MESSAGE_FORMAT AlarmerMessageFormat = "mkdown"
)

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
		defaultAlert      = AlertDefinition{}
	)

	pwd, err := os.Getwd()
	defaultAlertBytes, err = os.ReadFile(filepath.Join(pwd, "resources/default_alert.yaml"))
	if err != nil {
		log.Fatal(err)
	}

	err = yaml.Unmarshal(defaultAlertBytes, &defaultAlert)
	if err != nil {
		log.Fatal(err)
	}

	customAlertDefinitionPlace := os.Getenv(EnvAlertDefinitionPlace)

	customAlertDefinition := AlertDefinition{}
	if customAlertDefinitionPlace == "" {
	} else if strings.Contains(customAlertDefinitionPlace, "http") { // Request http
		client := http.Client{}
		req, err := requestGet(context.Background(), customAlertDefinitionPlace)
		if err != nil {
			return nil, err
		}

		res, err := request(client, req, 3)
		if err != nil {
			return nil, err
		}

		err = yaml.Unmarshal(res, &customAlertDefinition)
		if err != nil {
			return nil, err
		}

		for k, v := range customAlertDefinition.AlertLevel {
			defaultAlert.AlertLevel[k] = v
		}
		for k, v := range customAlertDefinition.Alarmer {
			defaultAlert.Alarmer[k] = v
		}

	} else { // Read filesystem

		customBytes, err := os.ReadFile(customAlertDefinitionPlace)
		if err != nil {
			return nil, err
		}

		err = yaml.Unmarshal(customBytes, &customAlertDefinitionPlace)
		if err != nil {
			return nil, err
		}

		for k, v := range customAlertDefinition.AlertLevel {
			defaultAlert.AlertLevel[k] = v
		}
		for k, v := range customAlertDefinition.Alarmer {
			defaultAlert.Alarmer[k] = v
		}
	}

	var (
		githubOwner = os.Getenv(EnvGithubOwner)
		repo        = os.Getenv(EnvGithubRepo)
		branch      = os.Getenv(EnvGithubBranch)
		githubPath  = os.Getenv(EnvGithubPath)
		githubToken = os.Getenv(EnvGithubToken)
		githubFile  []byte
	)
	if repo != "" {
		githubFile, err = util.FetchGithubFile(githubOwner, repo, branch, githubPath, githubToken)

		err = yaml.Unmarshal(githubFile, &customAlertDefinitionPlace)
		if err != nil {
			return nil, err
		}

		for k, v := range customAlertDefinition.AlertLevel {
			defaultAlert.AlertLevel[k] = v
		}
		for k, v := range customAlertDefinition.Alarmer {
			defaultAlert.Alarmer[k] = v
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

func requestGet(ctx context.Context, address string) (*http.Request, error) {
	return http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
}

func request(c http.Client, request *http.Request, retries int) ([]byte, error) {
	var errMsg string
	for i := 0; i < retries; i++ {
		res, err := c.Do(request)
		if err != nil {
			errMsg = errors.New("err: " + err.Error() + ", " + runtime.FuncForPC(reflect.ValueOf(request).Pointer()).Name() + ".Retries " + strconv.Itoa(i) + "...").Error()
			log.Warn(errMsg)
			continue
		}

		body, err := io.ReadAll(res.Body)
		if err != nil {
			errMsg = errors.New("err: " + err.Error() + ", " + runtime.FuncForPC(reflect.ValueOf(request).Pointer()).Name() + ".Retries " + strconv.Itoa(i) + "...").Error()
			log.Warn(errMsg)
			continue
		}
		defer res.Body.Close()

		return body, nil
	}

	return nil, errors.New(errMsg)
}

func ParseCheckerFunctions(defaultCheckerRegistry map[string]Func) []Checker {
	var result []Checker
	checkerFunctions := strings.Split(os.Getenv(EnvCheckerFunction), ",")
	for _, checkerName := range checkerFunctions {
		if checkerFunction, exists := defaultCheckerRegistry[checkerName]; exists {
			result = append(result, checkerFunction)
		}
	}

	if len(result) == 0 {
		for _, checkerFunction := range defaultCheckerRegistry {
			result = append(result, checkerFunction)
		}
	}
	return result
}
