package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/b-harvest/Harvestmon/repository"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func initializeViper() error {
	// Automatically bind environment variables
	viper.SetEnvPrefix("MONITOR") // All env vars should start with MONITOR
	viper.AutomaticEnv()

	// Bind environment variables to specific configuration keys
	var err error
	envVars := map[string]string{
		"agentName": "MONITOR_AGENT_NAME",
		"commitId":  "MONITOR_COMMIT_ID",

		"http.timeout":                       "MONITOR_HTTP_TIMEOUT",
		"http.transport.maxIdleConns":        "MONITOR_HTTP_MAX_IDLE_CONNS",
		"http.transport.maxIdleConnsPerHost": "MONITOR_HTTP_MAX_IDLE_CONNS_PER_HOST",
		"http.transport.idleConnTimeout":     "MONITOR_HTTP_IDLE_CONN_TIMEOUT",
		"http.transport.maxConnsPerHost":     "MONITOR_HTTP_MAX_CONN_PER_HOST",

		"database.user":            "MONITOR_DATABASE_USER",
		"database.password":        "MONITOR_DATABASE_PASSWORD",
		"database.host":            "MONITOR_DATABASE_HOST",
		"database.port":            "MONITOR_DATABASE_PORT",
		"database.dbName":          "MONITOR_DATABASE_NAME",
		"database.maxIdleConns":    "MONITOR_DATABASE_MAX_IDLE_CONNS",
		"database.maxOpenConns":    "MONITOR_DATABASE_MAX_OPEN_CONNS",
		"database.connMaxLifeTime": "MONITOR_DATABASE_CONN_MAX_LIFE_TIME",
		"database.connMaxIdleTime": "MONITOR_DATABASE_CONN_MAX_IDLE_TIME",

		"store.interval": "MONITOR_STORE_INTERVAL",
	}

	for key, env := range envVars {
		if err = viper.BindEnv(key, env); err != nil {
			return fmt.Errorf("error binding env var %s: %w", key, err)
		}
	}

	return nil
}

type Valid interface {
	Validate() error
}

type Config struct {
	ctx    context.Context
	cancel context.CancelFunc

	logger *log.Entry

	sharedHttpClient *http.Client

	Http *struct {
		Transport *struct {
			MaxIdleConns        int            `mapstructure:"maxIdleConns"`
			MaxIdleConnsPerHost int            `mapstructure:"maxIdleConnsPerHost"`
			IdleConnTimeout     *time.Duration `mapstructure:"idleConnTimeout"`
			MaxConnsPerHost     int            `mapstructure:"maxConnsPerHost"`
		} `mapstructure:"transport"`
		Timeout *time.Duration `mapstructure:"timeout"`
	} `mapstructure:"http"`

	CommitId string `mapstructure:"commitId"`

	AgentName string `mapstructure:"agentName"`

	TendermintMonitorConfigs map[MonitorTarget]*TendermintMonitorConfig `mapstructure:"tendermint"`
	EthereumMonitorConfigs   map[MonitorTarget]*EthereumMonitorConfig   `mapstructure:"ethereum"`
	NodeMonitorConfigs       map[MonitorTarget]*NodeMonitorConfig       `mapstructure:"node"`

	// Database stores connection information to connect with database.
	Database *Database `mapstructure:"database"`

	db *sql.DB

	// Store is repository holder.
	// it'll be pooling for while Database.DbBatchSize, and if the queue is full, execute insert command.
	Store *Store `mapstructure:"store"`
}

func (c *Config) getMonitorConfigs() []MonitorConfig {
	var monitorConfigs []MonitorConfig

	for _, tm := range c.TendermintMonitorConfigs {
		monitorConfigs = append(monitorConfigs, tm)
	}

	for _, eth := range c.EthereumMonitorConfigs {
		monitorConfigs = append(monitorConfigs, eth)
	}

	for _, node := range c.NodeMonitorConfigs {
		monitorConfigs = append(monitorConfigs, node)
	}

	return monitorConfigs
}

func (c *Config) setMonitorConfigs(mcs []MonitorConfig) {
	var (
		tcs = make(map[MonitorTarget]*TendermintMonitorConfig)
		ecs = make(map[MonitorTarget]*EthereumMonitorConfig)
		ncs = make(map[MonitorTarget]*NodeMonitorConfig)
	)
	for _, mc := range mcs {
		if tc, ok := mc.(*TendermintMonitorConfig); ok {
			tcs[tc.monitorTarget] = tc
		}
		if ec, ok := mc.(*EthereumMonitorConfig); ok {
			ecs[ec.monitorTarget] = ec
		}
		if nc, ok := mc.(*NodeMonitorConfig); ok {
			ncs[nc.monitorTarget] = nc
		}
	}

	c.TendermintMonitorConfigs = tcs
	c.EthereumMonitorConfigs = ecs
	c.NodeMonitorConfigs = ncs
}

func (c *Config) Validate() error {
	if c.logger == nil {
		return errors.New("logger is required")
	}

	if c.CommitId == "" {
		return errors.New("commitId is required")
	}

	if c.AgentName == "" {
		return errors.New("agentName is required")
	}

	for _, mc := range c.getMonitorConfigs() {
		err := mc.Validate()
		if err != nil {
			return errors.Wrapf(err, "validating monitor config: %v", mc.getMonitorName())
		}
	}

	if c.db == nil {
		return errors.New("cannot connect to database")
	}

	if err := c.Database.Validate(); err != nil {
		return errors.Wrap(err, "validating database config")
	}

	return nil
}

type StoreEntity struct {
	tmCommit  *repository.TendermintCommit
	tmNetInfo *repository.TendermintNetInfo
	tmStatus  *repository.TendermintStatus
	ethBlock  *repository.EthereumBlockNumber
}

type MonitorTarget string

type MonitorConfig interface {
	Validate() error

	initialize(
		agentName, commitId string,
		batchSize int,
		logger *log.Entry,
		client *http.Client,
		collectors []Collector)

	getMonitorName() string
	getLogger() *log.Entry

	getCollectors() []Collector

	load(r *repository.BaseRepository) error
	exit(r *repository.BaseRepository) error
}

var CollectorFuncRegistry = map[string]CollectorFunc{
	"tm:status":        tmStatusCollector,
	"tm:net_info":      tmNetInfoCollector,
	"tm:commit":        tmCommitCollector,
	"eth:block_number": ethBlockNumberCollector,
}

// Collector is the function for collect metrics, and send it to storeEntityChan.
type Collector struct {
	Name     string         `mapstructure:"name"`
	Interval *time.Duration `mapstructure:"interval"`
	CollectorFunc

	logger *log.Entry
}

func (c *Collector) Validate() error {
	if c.Name == "" {
		return errors.New("collector name is required")
	}
	if c.Interval == nil {
		return errors.New("collector interval is required")
	}
	return nil
}

func (c *Collector) setLogger(logger *log.Entry) {
	c.logger = logger
}

type CollectorFunc func(MonitorConfig) func(sq chan StoreEntity, l *log.Entry)

func (c *Config) SaveOnExit(saved chan interface{}) {

	quitting := make(chan os.Signal, 1)
	signal.Notify(quitting, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	saveState := func() {
		defer close(saved)
		log.Info("saving state...")

		for _, mc := range c.getMonitorConfigs() {
			err := mc.exit(c.Store.repo)
			if err != nil {
				c.logger.Error(err.Error())
			}
		}
		c.Store.store()

		log.Info("monitor exiting.")
	}
	for {
		select {
		case <-c.ctx.Done():
			saveState()
			return
		case <-quitting:
			saveState()
			c.cancel()
			return
		}
	}
}

func request(c *http.Client, request *http.Request, retries int) ([]byte, error) {
	var (
		err  error
		res  *http.Response
		body []byte
	)
	for i := 0; i < retries; i++ {
		res, err = c.Do(request)
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}

		body, err = io.ReadAll(res.Body)
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}
		defer res.Body.Close()

		return body, nil
	}

	return nil, err
}

func getEndpoint(host string, port int) string {
	prefix := "http"
	if port == 443 {
		prefix = "https"
	}
	return fmt.Sprintf("%s://%s:%d", prefix, host, port)
}
