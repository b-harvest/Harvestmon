package main

import (
	"errors"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"os"
	"strings"
	"sync"
	"time"
)

// LoadConfig initializes the configuration and sets up watching for changes.
func (am *AlertManager) LoadConfig(configPath string) error {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("toml")

	setGithubDefaults(viper.GetViper())

	if err := viper.ReadInConfig(); err != nil {
		return err
	}

	// Parse main configuration
	var config AlertManager
	if err := viper.Unmarshal(&config); err != nil {
		return err
	}
	// initialize CheckerConfig logger
	config.logger = am.logger
	config.mu = &sync.RWMutex{}

	if config.GithubFile != nil {
		if path := config.GithubFile.Path; path != "" {
			var alrmerCfgBytes [][]byte
			if alrmerCfgBytes = config.GithubFile.getFilesBytes(config.GithubFile.Path); len(alrmerCfgBytes) == 0 {
				return errors.New("alert config file is empty")
			}

			v := viper.New()
			v.SetConfigType("toml")

			// Expand ${ENV}/$ENV references (e.g. secrets like botToken/apiKey) so that
			// credentials can be injected via environment variables instead of being
			// committed in plaintext to the GitHub-hosted alarmer config file.
			expanded := os.ExpandEnv(string(alrmerCfgBytes[0]))

			if err := v.ReadConfig(strings.NewReader(expanded)); err != nil {
				return errors.New("error reading alert config file" + err.Error())
			}

			var alrmerCfg AlarmerConfig
			if err := v.Unmarshal(&alrmerCfg); err != nil {
				return errors.New("error unmarshalling alert config file" + err.Error())
			}
			config.AlarmerConfig = &alrmerCfg
		} else {
			return errors.New("alert manager config file path is empty")
		}
	}

	setDatabaseDefaults(&config.RDatabaseCfg)
	setDatabaseDefaults(&config.WDatabaseCfg)

	// Update the config in the manager
	if am.writer != nil {
		config.writer = am.writer
	} else {
		wdb, err := GetDatabase(config.WDatabaseCfg)
		if err != nil {
			return errors.New(fmt.Sprintf("Error loading write db connection: %s", err))
		}

		config.writer, err = newRepository(wdb)
		if err != nil {
			return errors.New(fmt.Sprintf("Error loading repository: %v", err))
		}
	}

	if am.reader != nil {
		config.reader = am.reader
	} else {
		rdb, err := GetDatabase(config.RDatabaseCfg)
		if err != nil {
			return errors.New(fmt.Sprintf("Error loading rdb connection: %s", err))
		}

		config.reader, err = newRepository(rdb)
		if err != nil {
			return errors.New(fmt.Sprintf("Error loading repository: %v", err))
		}
	}

	am.replace(&config)

	// Set up a watcher for changes
	go am.watchConfig()

	return nil
}

func setDatabaseDefaults(dbConfig *Database) {
	if dbConfig.maxIdleConns == 0 {
		dbConfig.maxIdleConns = 5
	}
	if dbConfig.maxOpenConns == 0 {
		dbConfig.maxOpenConns = 5
	}
	if dbConfig.connMaxIdleTime == time.Duration(0) {
		dbConfig.connMaxIdleTime = 1 * time.Minute
	}
}

func setGithubDefaults(v *viper.Viper) {
	v.SetDefault("github.branch", "main")
}

// watchConfig watches the configuration file and reloads it on changes.
func (am *AlertManager) watchConfig() {
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		am.logger.Printf("Configuration file changed: %s", e.Name)
		err := am.LoadConfig(viper.ConfigFileUsed())
		if err != nil {
			am.logger.Printf("Error reloading configuration: %v", err)
		} else {
			am.logger.Println("Configuration reloaded successfully")
		}
	})
}

// getConfig returns a copy of the current configuration for safe access.
func (am *AlertManager) getConfig() error {
	var (
		err error
	)
	if am.GithubFile.isChanged() {
		if err = am.LoadConfig(viper.ConfigFileUsed()); err != nil {
			return err
		}
	}

	if err = am.checkValid(); err != nil {
		return err
	}

	return nil
}

func (am *AlertManager) replace(newAm *AlertManager) {
	am.mu.Lock() // Ensure thread-safety when replacing fields
	defer am.mu.Unlock()

	if newAm.logger != nil {
		am.logger = newAm.logger
	}
	if newAm.WDatabaseCfg != (Database{}) {
		am.WDatabaseCfg = newAm.WDatabaseCfg
	}
	if newAm.RDatabaseCfg != (Database{}) {
		am.RDatabaseCfg = newAm.RDatabaseCfg
	}
	if newAm.GithubFile != nil {
		am.GithubFile = newAm.GithubFile
	}

	if newAm.AlarmerConfig != nil {
		am.AlarmerConfig = newAm.AlarmerConfig
	}
	if newAm.reader != nil {
		am.reader = newAm.reader
	}
	if newAm.writer != nil {
		am.writer = newAm.writer
	}

	if newAm.record != nil {
		recordCopy := make(map[InstanceName]map[TargetName]*Record)
		for instanceName, targets := range newAm.record {
			targetCopy := make(map[TargetName]*Record)
			for targetName, record := range targets {
				targetCopy[targetName] = &Record{
					mtx:         &sync.Mutex{}, // Create a new mutex for the copied record
					activeAlert: make(map[alertEvent]time.Time),
					alarmCache:  make(map[alarmName]int64),
				}
				for alert, timeVal := range record.activeAlert {
					targetCopy[targetName].activeAlert[alert] = timeVal
				}
				for alarm, count := range record.alarmCache {
					targetCopy[targetName].alarmCache[alarm] = count
				}
			}
			recordCopy[instanceName] = targetCopy
		}
		am.record = recordCopy
	}
}
