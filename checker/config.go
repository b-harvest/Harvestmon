package main

import (
	"bytes"
	"errors"
	"fmt"
	_const "github.com/b-harvest/Harvestmon/const"
	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
	"sync"
	"time"
)

type ConfigManager struct {
	logger *log.Entry

	config CheckerConfig
	mu     sync.RWMutex
}

// LoadConfig initializes the configuration and sets up watching for changes.
func (cm *ConfigManager) LoadConfig(configPath string) error {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("toml")

	setGithubDefaults(viper.GetViper())

	if err := viper.ReadInConfig(); err != nil {
		return err
	}

	// Parse main configuration
	var config CheckerConfig
	if err := viper.Unmarshal(&config); err != nil {
		return err
	}

	setDatabaseDefaults(&config.RDatabaseCfg)
	setDatabaseDefaults(&config.WDatabaseCfg)

	// initialize CheckerConfig logger
	config.logger = cm.logger.WithField("commitId", config.CommitId)

	// Update the config in the manager
	cm.mu.Lock()
	if cm.config.writeRepo != nil {
		config.writeRepo = cm.config.writeRepo
	} else {
		wdb, err := GetDatabase(config.WDatabaseCfg)
		if err != nil {
			return errors.New(fmt.Sprintf("Error loading write db connection: %s", err))
		}

		config.writeRepo, err = config.getRepository(wdb)
		if err != nil {
			return errors.New(fmt.Sprintf("Error loading repository: %v", err))
		}

	}

	if cm.config.readRepo != nil {
		config.readRepo = cm.config.readRepo
	} else {
		rdb, err := GetDatabase(config.RDatabaseCfg)
		if err != nil {
			return errors.New(fmt.Sprintf("Error loading rdb connection: %s", err))
		}

		config.readRepo, err = config.getRepository(rdb)
		if err != nil {
			return errors.New(fmt.Sprintf("Error loading repository: %v", err))
		}

	}

	cm.config = config
	cm.mu.Unlock()

	// Alarmer setup
	// it'll read config file from github
	if path := config.GithubFile.AlarmerConfigPath; path != "" {
		var checkerAlarmerConfigBytes [][]byte
		if checkerAlarmerConfigBytes = config.GithubFile.getFilesBytes(config.GithubFile.AlarmerConfigPath); len(checkerAlarmerConfigBytes) == 0 {
			return errors.New("checker config file is empty")
		}

		v := viper.GetViper()
		v.SetConfigType("toml")

		if err := v.ReadConfig(bytes.NewReader(checkerAlarmerConfigBytes[0])); err != nil {
			return errors.New("error reading checker config file" + err.Error())
		}

		var checkerAlarmerConfig AlarmerConfig
		if err := v.Unmarshal(&checkerAlarmerConfig); err != nil {
			return errors.New("error unmarshalling checker config file" + err.Error())
		}
		config.AlarmerConfig = checkerAlarmerConfig
	}

	var nodeInfos []NodeInfo

	//
	// NodeInfos setup
	// 1). local files
	// 2). remote files
	//
	nodeInfoPaths := viper.GetStringSlice("node_infos")
	for _, path := range nodeInfoPaths {
		if path == "" {
			continue
		}

		content, err := os.ReadFile(path)
		if err != nil {
			cm.logger.Warningf("failed to read node info from config file %s, err: %v", path, err)
			continue
		}

		viper.SetConfigType("toml")
		if err = viper.ReadConfig(bytes.NewReader(content)); err != nil {
			cm.logger.Warningf("failed to parse node info from config file %s, err: %v", path, err)
			continue
		}
		var node NodeInfo
		if err = viper.Unmarshal(&node); err != nil {
			cm.logger.Warningf("failed to unmarshal node info from config file, err: %v", err)
			continue
		}

		// initialize node logger
		node.logger = config.logger.WithField("node", node.Name)
		nodeInfos = append(nodeInfos, node)
	}

	if len(config.GithubFile.NodeInfoPaths) == 0 {
		config.GithubFile.NodeInfoPaths = append(config.GithubFile.NodeInfoPaths, "/")
	}

	// processing on goroutine
	var (
		nodeBytesMtx sync.RWMutex
		wg           sync.WaitGroup
	)
	for _, nodeInfoFilePath := range config.GithubFile.NodeInfoPaths {
		wg.Add(1)
		go func() {
			defer func() {
				wg.Done()
			}()
			nodesBytes := config.GithubFile.getFilesBytes(nodeInfoFilePath)

			for _, nodeBytes := range nodesBytes {
				wg.Add(1)
				go func(nb []byte) {
					defer wg.Done()
					v := viper.New()
					v.SetConfigType("toml")
					if err := v.ReadConfig(bytes.NewReader(nb)); err != nil {
						cm.logger.Warningf("failed to parse node info from config file %s, err: %v", nodeInfoFilePath, err)
						return
					}
					var node NodeInfo
					if err := v.Unmarshal(&node); err != nil {
						cm.logger.Warningf("failed to unmarshal node info from config file %s, err: %v", nodeInfoFilePath, err)
						return
					}

					// initialize node logger
					node.logger = config.logger.WithField("node", node.Name)

					if node.Alarmer == nil {
						node.Alarmer = &config.AlarmerConfig
					}

					for _, t := range node.Alerts.Tendermint {
						if t == nil {
							continue
						}

						t.logger = node.logger.WithField("service", _const.HARVESTMON_TENDERMINT_SERVICE_NAME)

						for _, strategy := range t.getAlertStrategies() {
							if strategy == nil {
								continue
							}
							strategy.initialize(
								t.logger.WithField("strategy", strategy.getName()),
								config.readRepo,
							)
							t.logger.Debug("set alert strategy: " + strategy.getName())
						}

					}
					for _, e := range node.Alerts.Ethereum {
						if e == nil {
							continue
						}
						e.logger = node.logger.WithField("service", _const.HARVESTMON_ETHEREUM_SERVICE_NAME)

						for _, strategy := range e.getAlertStrategies() {
							if strategy == nil {
								continue
							}
							strategy.initialize(
								e.logger.WithField("strategy", strategy.getName()),
								config.readRepo,
							)
							e.logger.Debug("set alert strategy: " + strategy.getName())
						}
					}

					nodeBytesMtx.Lock()
					nodeInfos = append(nodeInfos, node)
					nodeBytesMtx.Unlock()
				}(nodeBytes)
			}

		}()
	}
	wg.Wait()

	config.NodeInfos = nodeInfos
	cm.config = config
	// Set up a watcher for changes
	go cm.watchConfig()

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
func (cm *ConfigManager) watchConfig() {
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		cm.logger.Printf("Configuration file changed: %s", e.Name)
		err := cm.LoadConfig(viper.ConfigFileUsed())
		if err != nil {
			cm.logger.Printf("Error reloading configuration: %v", err)
		} else {
			cm.logger.Println("Configuration reloaded successfully")
		}
	})
}

// GetConfig returns a copy of the current configuration for safe access.
func (cm *ConfigManager) GetConfig() (*CheckerConfig, error) {
	if cm.config.GithubFile.isChanged() {
		if err := cm.LoadConfig(viper.ConfigFileUsed()); err != nil {
			return nil, err
		}
	}

	cm.mu.RLock()
	defer cm.mu.RUnlock()
	if err := cm.config.checkValid(); err != nil {
		return nil, err
	}

	return &cm.config, nil
}
