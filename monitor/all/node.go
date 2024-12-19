package main

import (
	_const "github.com/b-harvest/Harvestmon/const"
	"github.com/b-harvest/Harvestmon/repository"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net/http"
)

type NodeMonitorConfig struct {
	logger *log.Entry

	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`

	agentName string
	commitId  string

	monitorTarget MonitorTarget
	Collectors    []Collector `mapstructure:"collectors"`

	client    *http.Client
	batchSize int
}

func (c *NodeMonitorConfig) Validate() error {
	if c.logger == nil {
		return errors.New("logger is required")
	}

	if c.Host == "" {
		return errors.New("host is required")
	}

	if c.Port == 0 {
		return errors.New("port is required")
	}

	if c.agentName == "" {
		return errors.New("agent name is required")
	}

	if c.commitId == "" {
		return errors.New("commitId is required")
	}

	if len(c.Collectors) == 0 {
		return errors.New("collectors is required")
	}

	if c.batchSize == 0 {
		return errors.New("batchSize is required")
	}

	if c.client == nil {
		return errors.New("client is required")
	}

	for _, collector := range c.Collectors {
		if err := collector.Validate(); err != nil {
			return errors.Wrapf(err, "collector '%v' is invalid", collector.Name)
		}
	}
	return nil
}

func (c *NodeMonitorConfig) initialize(
	agentName, commitId string,
	batchSize int,
	logger *log.Entry,
	client *http.Client,
	collectors []Collector) {

	c.agentName = agentName
	c.commitId = commitId
	c.batchSize = batchSize
	c.logger = logger
	c.client = client
	c.Collectors = collectors
}

func (c *NodeMonitorConfig) getMonitorName() string {
	return _const.HARVESTMON_NODE_SERVICE_NAME
}

func (c *NodeMonitorConfig) getLogger() *log.Entry {
	return c.logger
}

func (c *NodeMonitorConfig) getCollectors() []Collector {
	return c.Collectors
}

func (c *NodeMonitorConfig) load(r *repository.BaseRepository) error {
	return nil
}

func (c *NodeMonitorConfig) exit(repo *repository.BaseRepository) error {
	return nil
}

//
//var nodeCollector = func(mc MonitorConfig) func(sq chan StoreEntity, l *log.Entry) {
//	c, ok := mc.(*NodeMonitorConfig)
//	if !ok {
//		return func(sq chan StoreEntity, l *log.Entry) {
//			l.Warningf("invalid config type")
//			return
//		}
//	}
//
//	return func(sq chan StoreEntity, l *log.Entry) {
//
//		endpoint := getEndpoint(c.Host, c.Port)
//		endpoint += "/metrics"
//
//		ctx, cancel := context.WithTimeout(context.Background(), c.client.Timeout)
//		defer cancel()
//		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
//		if err != nil {
//			l.Warningf(errors.Wrapf(err, "failed to build request '%s'", endpoint).Error())
//			return
//		}
//
//		var (
//			body         []byte
//			statusResult CometBFTStatusResult
//		)
//		body, err = request(c.client, req, 3)
//		if err != nil {
//			l.Warningf(errors.Wrapf(err, "failed to request to endpoint").Error())
//			return
//		}
//
//		err = json.Unmarshal(body, &statusResult)
//		if err != nil {
//			l.Warningf(err.Error())
//			return
//		}
//
//		tmStatus, err := newTendermintStatus(c.agentName, c.commitId, statusResult)
//		if err != nil {
//			l.Error(err.Error())
//			return
//		}
//		sq <- StoreEntity{
//			tmStatus: tmStatus,
//		}
//
//		c.logger.Debugf("got tendermint status height: %v", tmStatus.LatestBlockHeight)
//		c.height = tmStatus.LatestBlockHeight
//
//		return
//	}
//}
