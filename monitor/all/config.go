package main

import (
	"context"
	"github.com/b-harvest/Harvestmon/repository"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"net/http"
	"time"
)

var (
	DefaultCollectorInterval = time.Second * 5
	DefaultStoreTimeout      = time.Second * 5
	DefaultDbBatchSize       = 100
)

func loadConfig(configPath string) error {
	var err error

	viper.SetConfigFile(configPath)
	viper.SetConfigType("toml")

	if err = viper.ReadInConfig(); err != nil {
		return err
	}

	if err = viper.Unmarshal(&cfg); err != nil {
		return err
	}

	if cfg.Database == nil {
		return errors.New("database config is required")
	}
	setDatabaseDefaults(cfg.Database)

	cfg.ctx, cfg.cancel = context.WithCancel(context.Background())

	var (
		logKeyMonitor   = "monitor"
		logKeyCollector = "collector"
		//logKeyMonitorTarget = "target"
	)

	cfg.sharedHttpClient = &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        50,
			MaxIdleConnsPerHost: 5,
			MaxConnsPerHost:     10,
			IdleConnTimeout:     30 * time.Second,
		},
		Timeout: 30 * time.Second,
	}

	if cfg.Http != nil {
		if cfg.Http.Timeout != nil {
			cfg.sharedHttpClient.Timeout = *cfg.Http.Timeout
		}
		if cfg.Http.Transport != nil {
			newTransport := &http.Transport{}

			t := cfg.Http.Transport
			if t.MaxIdleConns != 0 {
				newTransport.MaxIdleConns = t.MaxIdleConns
			}
			if t.MaxIdleConnsPerHost != 0 {
				newTransport.MaxIdleConnsPerHost = t.MaxIdleConnsPerHost
			}
			if t.IdleConnTimeout != nil {
				newTransport.IdleConnTimeout = *t.IdleConnTimeout
			}
			if t.MaxConnsPerHost != 0 {
				newTransport.MaxConnsPerHost = t.MaxConnsPerHost
			}
			cfg.sharedHttpClient.Transport = newTransport
		}
	}

	if cfg.db, err = GetDatabase(cfg.Database); err != nil {
		return err
	}

	var repo *repository.BaseRepository
	if repo, err = cfg.getRepository(); err != nil {
		return err
	}

	storeInterval := &DefaultStoreTimeout
	if cfg.Store != nil && cfg.Store.Interval != nil {
		storeInterval = cfg.Store.Interval
	}

	cfg.Store = &Store{
		logger:      cfg.logger.WithFields(log.Fields{}).Logger,
		repo:        repo,
		poolingSize: cfg.Database.DbBatchSize,
		Interval:    storeInterval,
	}

	cfg.logger = cfg.logger.WithField("commitId", cfg.CommitId)

	var (
		monitorConfigs []MonitorConfig
	)
	for _, mc := range cfg.getMonitorConfigs() {
		var collectors []Collector
		for _, collector := range mc.getCollectors() {
			if collector.Name == "" {
				continue
			}
			f, ok := CollectorFuncRegistry[collector.Name]
			if !ok {
				cfg.logger.Warningf("collector %s not registered", collector.Name)
				continue
			}

			collector.setLogger(cfg.logger.WithFields(log.Fields{
				logKeyCollector: collector.Name,
			}))
			collector.CollectorFunc = f

			if collector.Interval == nil {
				collector.Interval = &DefaultCollectorInterval
			}

			collectors = append(collectors, collector)
		}
		if len(collectors) == 0 {
			cfg.logger.Warningf("no collectors defined in config: %v", mc.getMonitorName())
			continue
		}

		mc.initialize(
			cfg.AgentName, cfg.CommitId,
			cfg.Database.DbBatchSize,
			cfg.logger.WithFields(log.Fields{
				logKeyMonitor: mc.getMonitorName(),
			}),
			cfg.sharedHttpClient,
			collectors,
		)

		err = mc.load(cfg.Store.repo)
		if err != nil {
			cfg.logger.WithError(err).Error("failed to load monitor")
			continue
		}

		monitorConfigs = append(monitorConfigs, mc)
	}

	cfg.setMonitorConfigs(monitorConfigs)

	if err = cfg.Validate(); err != nil {
		return err
	}

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
	if dbConfig.DbBatchSize == 0 {
		dbConfig.DbBatchSize = DefaultDbBatchSize
	}
}
