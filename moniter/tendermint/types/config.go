package types

import (
	"errors"
	"fmt"
	"github.com/b-harvest/Harvestmon/log"
	"os"
	"strconv"
	"strings"
	"time"
)

type MonitorConfig struct {
	Agent       MonitoringAgent `yaml:"agent"`
	DbBatchSize int             `yaml:"dbBatchSize"`
}

type MonitoringAgent struct {
	AgentName                 string         `yaml:"name"`
	Host                      string         `yaml:"host"`
	Port                      int            `yaml:"port"`
	Monitors                  []Func         `yaml:"monitors"`
	PushInterval              *time.Duration `yaml:"pushInterval"`
	BlockCommitMaxConcurrency int            `yaml:"blockCommitMaxConcurrency"`
	Timeout                   *time.Duration `yaml:"timeout"`
	CommitId                  string         `yaml:"commitId"`
}

var (
	EnvTimeout                   = "TIMEOUT"
	EnvAgentName                 = "AGENT_NAME"
	EnvAgentHost                 = "AGENT_HOST"
	EnvAgentPort                 = "AGENT_PORT"
	EnvPushInterval              = "PUSH_INTERVAL"
	EnvBlockCommitMaxConcurrency = "BLOCK_COMMIT_MAX_CONCURRENCY"
	EnvMonitors                  = "AGENT_MONITORS"
	EnvCommitId                  = "COMMIT_ID"

	EnvConfigFilePath = "CONFIG_FILE_PATH"
)

var (
	DefaultTimeout                   = 3 * time.Second
	DefaultAgentName                 = "instance"
	DefaultAgentHost                 = "127.0.0.1"
	DefaultAgentPort                 = 26657
	DefaultPushInterval              = 10 * time.Second
	DefaultBlockCommitMaxConcurrency = 100
)

var MonitorRegistry map[string]Func

func (f *Func) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Temporary structure to unmarshal the YAML into
	var tmp struct {
		Name     string         `yaml:"name"`
		Interval *time.Duration `yaml:"interval"`
	}

	// Unmarshal into the temporary struct
	if err := unmarshal(&tmp); err != nil {
		return err
	}

	// Look up the MonitorFunc based on the name
	monitor, exists := MonitorRegistry[tmp.Name]
	if !exists {
		return fmt.Errorf("unknown monitor: %s", tmp.Name)
	}

	// Assign the found MonitorFunc and Interval to the Func struct
	f.MonitorFunc = monitor.MonitorFunc
	f.Interval = tmp.Interval

	return nil
}

// ApplyConfigFromEnvAndDefault will read the environmental variables into a config
// then validate it is reasonable and if there are not set in any column, set as defaults.
func (cfg *MonitorConfig) ApplyConfigFromEnvAndDefault() error {

	if cfg.Agent.Timeout == nil {
		v := os.Getenv(EnvTimeout)
		if v == "" {
			cfg.Agent.Timeout = &DefaultTimeout
			log.Debug("timeout set as default: " + cfg.Agent.Timeout.String())
		} else {
			timeout, err := parseEnvDuration(v)
			if err != nil {
				return errors.New(err.Error())
			}
			cfg.Agent.Timeout = &timeout
			log.Debug("timeout set as ENV: " + cfg.Agent.Timeout.String())
		}
	} else {
		log.Debug("timeout set as " + cfg.Agent.Timeout.String())
	}

	if cfg.Agent.PushInterval == nil {
		v := os.Getenv(EnvPushInterval)
		if v == "" {
			cfg.Agent.PushInterval = &DefaultPushInterval
			log.Debug("pushInterval set as default: " + cfg.Agent.PushInterval.String())
		} else {
			interval, err := parseEnvDuration(v)
			if err != nil {
				return errors.New(err.Error())
			}
			cfg.Agent.PushInterval = &interval
			log.Debug("pushInterval set as ENV: " + cfg.Agent.PushInterval.String())
		}
	} else {
		log.Debug("pushInterval set as " + cfg.Agent.PushInterval.String())
	}

	if cfg.Agent.BlockCommitMaxConcurrency == 0 {
		v := os.Getenv(EnvBlockCommitMaxConcurrency)
		if v == "" {
			cfg.Agent.BlockCommitMaxConcurrency = DefaultBlockCommitMaxConcurrency
			log.Debug("pushInterval set as default: " + cfg.Agent.PushInterval.String())
		} else {
			concurrency, err := strconv.Atoi(v)
			if err != nil {
				log.Error(errors.New("error occurred while parsing blockCommitMaxConcurrency " + err.Error()))
				concurrency = DefaultBlockCommitMaxConcurrency
			}
			cfg.Agent.BlockCommitMaxConcurrency = concurrency
			log.Debug("blockCommitMaxConcurrency set as ENV: " + strconv.Itoa(cfg.Agent.BlockCommitMaxConcurrency))
		}

	}

	if cfg.Agent.AgentName == "" {
		v := os.Getenv(EnvAgentName)
		if v == "" {
			log.Warn(errors.New("Could not found agent(node)'s agentName. it'll be set as `instance` temporarily. \n" +
				"You should set node's agentName as fast as possible. it may cause confusion.").Error())
			cfg.Agent.AgentName = DefaultAgentName
		} else {
			cfg.Agent.AgentName = v
			log.Debug("agentName set as ENV: " + cfg.Agent.AgentName)
		}
	} else {
		log.Debug("agentName set as " + cfg.Agent.AgentName)
	}

	if cfg.Agent.Host == "" {
		v := os.Getenv(EnvAgentHost)
		if v == "" {
			cfg.Agent.Host = DefaultAgentHost
			log.Debug("host set as default: " + cfg.Agent.Host)
		} else {
			cfg.Agent.Host = v
			log.Debug("host set as ENV: " + cfg.Agent.Host)
		}
	} else {
		log.Debug("host set as " + cfg.Agent.Host)
	}

	if cfg.Agent.Port == 0 {
		v := os.Getenv(EnvAgentPort)
		if v == "" {
			cfg.Agent.Port = DefaultAgentPort
			log.Debug("port set as " + strconv.Itoa(cfg.Agent.Port))
		} else {
			port, err := strconv.Atoi(v)
			if err != nil {
				return errors.New(err.Error())
			}
			cfg.Agent.Port = port
			log.Debug("port set as ENV" + strconv.Itoa(cfg.Agent.Port))
		}
	} else {
		log.Debug("port set as " + strconv.Itoa(cfg.Agent.Port))
	}

	if len(cfg.Agent.Monitors) == 0 {
		v := os.Getenv(EnvMonitors)
		if v == "" {
			for _, monFunc := range MonitorRegistry {
				cfg.Agent.Monitors = append(cfg.Agent.Monitors, monFunc)
			}
		} else {
			for _, name := range strings.Split(v, ",") {
				monitorFunc, exists := MonitorRegistry[name]
				if !exists {
					return errors.New("unknown service: " + name)
				}
				cfg.Agent.Monitors = append(cfg.Agent.Monitors, monitorFunc)
			}
			log.Debug("monitors set as " + v)
		}
	}

	if cfg.Agent.CommitId == "" {
		v := os.Getenv(EnvCommitId)
		if v == "" {
			return errors.New("No commit id found. please set commit id through config.yaml or env($COMMIT_ID)")
		}
		cfg.Agent.CommitId = v
		log.Debug("CommitId set as ENV: " + cfg.Agent.CommitId)
	} else {
		log.Debug("CommitId set as " + cfg.Agent.CommitId)
	}

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
