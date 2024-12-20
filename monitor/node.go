package main

import (
	"bytes"
	"context"
	"fmt"
	_const "github.com/b-harvest/Harvestmon/const"
	"github.com/b-harvest/Harvestmon/repository"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	NODE_MONITOR_CPU_KEY     = "node:cpu"
	NODE_MONITOR_DISK_KEY    = "node:disk"
	NODE_MONITOR_MEMORY_KEY  = "node:memory"
	NODE_MONITOR_NETWORK_KEY = "node:network"
	NODE_MONITOR_SYSTEMD_KEY = "node:systemd"
)

type NodeMonitorConfig struct {
	logger *log.Entry

	Host     string         `mapstructure:"host"`
	Port     int            `mapstructure:"port"`
	Interval *time.Duration `mapstructure:"interval"`

	agentName string
	commitId  string

	monitorTarget MonitorTarget
	Collectors    []Collector `mapstructure:"collectors"`

	client    *http.Client
	batchSize int

	querierMtx sync.Mutex

	watchMetricKeys []string
}

func (c *NodeMonitorConfig) getInterval() *time.Duration {
	return c.Interval
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

	mostShortInterval := time.Hour * 24
	var collectorName = ""
	for idx, collector := range collectors {
		if mostShortInterval > *collector.Interval {
			mostShortInterval = *collector.Interval
		}
		c.watchMetricKeys = append(c.watchMetricKeys, collector.Name)

		collectorName += fmt.Sprintf("%s", collector.Name)
		if idx != len(collectors)-1 {
			collectorName += ","
		}
	}

	c.Collectors = []Collector{
		{
			Name:          collectorName,
			Interval:      &mostShortInterval,
			CollectorFunc: nodeCollector,
			logger:        logger,
		},
	}

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

func (c *NodeMonitorConfig) load(r *repository.Repository) error {
	return nil
}

func (c *NodeMonitorConfig) exit(repo *repository.Repository) error {
	return nil
}

const (
	// Disk
	NODE_METRIC_FS_FREE_BYTES = "node_filesystem_free_bytes"

	// CPU
	NODE_METRIC_CPU_SECONDS_TOTAL = "node_cpu_seconds_total"

	// Memory
	NODE_MEM_TOTAL_BYTES_KEY       = "node_memory_MemTotal_bytes"
	NODE_MEM_FREE_BYTES_KEY        = "node_memory_MemFree_bytes"
	NODE_MEM_BUFFER_BYTES_KEY      = "node_memory_Buffers_bytes"
	NODE_MEM_CACHED_BYTES_KEY      = "node_memory_Cached_bytes"
	NODE_MEM_SLAB_BYTES_KEY        = "node_memory_Slab_bytes"
	NODE_MEM_PAGETABLES_BYTES_KEY  = "node_memory_PageTables_bytes"
	NODE_MEM_SWAP_CACHED_BYTES_KEY = "node_memory_SwapCached_bytes"

	// Systemd
	NODE_SYSTEMD_UNIT_STATE_KEY = "node_systemd_unit_state"

	// Network
	NODE_NETWORK_TRANSMIT_TOTAL_KEY = "node_network_transmit_bytes_total"
	NODE_NETWORK_RECEIVE_TOTAL_KEY  = "node_network_receive_bytes_total"
)

var nodeCollector = func(mc MonitorConfig) func(sq chan repository.StoreEntity, l *log.Entry) {
	c, ok := mc.(*NodeMonitorConfig)
	if !ok {
		return func(sq chan repository.StoreEntity, l *log.Entry) {
			l.Warningf("invalid config type")
			return
		}
	}

	return func(sq chan repository.StoreEntity, l *log.Entry) {
		l.Infof("node monitor config %v", c.Host)

		endpoint := getEndpoint(c.Host, c.Port)
		endpoint += "/metrics"

		ctx, cancel := context.WithTimeout(context.Background(), c.client.Timeout)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			l.Warningf(errors.Wrapf(err, "failed to build request '%s'", endpoint).Error())
			return
		}

		var (
			body  []byte
			mfMap map[string]*dto.MetricFamily
		)
		body, err = request(c.client, req, 3)
		if err != nil {
			l.Warningf(errors.Wrapf(err, "failed to request to endpoint").Error())
			return
		}

		mfMap, err = parseMF(body)

		if err != nil {
			l.Warningf(err.Error())
			return
		}

		var nodeMonitorFuncs = map[string]func(agentName, commitId string, mfMap map[string]*dto.MetricFamily) ([]repository.StoreEntity, error){
			NODE_MONITOR_CPU_KEY:     newNodeCPUSecondsTotal,
			NODE_MONITOR_DISK_KEY:    newNodeFreeDiskStatus,
			NODE_MONITOR_MEMORY_KEY:  newNodeMemoryStatus,
			NODE_MONITOR_SYSTEMD_KEY: newNodeSystemdStatus,
			NODE_MONITOR_NETWORK_KEY: newNodeNetworkTotal,
		}

		for _, watch := range c.watchMetricKeys {
			go func() {
				storeEntities, err := nodeMonitorFuncs[watch](c.agentName, c.commitId, mfMap)
				if err != nil {
					l.Warningf(errors.Wrapf(err, "failed to watch metrics").Error())
					return
				}
				for _, storeEntity := range storeEntities {
					sq <- storeEntity
				}
			}()
		}

		return
	}
}

func parseMF(content []byte) (map[string]*dto.MetricFamily, error) {

	var parser expfmt.TextParser
	mf, err := parser.TextToMetricFamilies(bytes.NewBuffer(content))
	if err != nil {
		return nil, err
	}
	return mf, nil
}

func newNodeFreeDiskStatus(agentName, commitId string, mfMap map[string]*dto.MetricFamily) ([]repository.StoreEntity, error) {
	var (
		freeDiskStatuses []repository.StoreEntity
		m                *dto.MetricFamily
		exists           bool
	)

	if m, exists = mfMap[NODE_METRIC_FS_FREE_BYTES]; !exists {
		return nil, errors.New("node free disk status not exists")
	}

	for _, metric := range m.Metric {
		var (
			device       string
			mountpoint   string
			freeDiskSize uint64
		)

		freeDiskSize = uint64(metric.GetGauge().GetValue())
		for _, label := range metric.GetLabel() {
			if label.GetName() == "device" {
				device = label.GetValue()
			} else if label.GetName() == "mountpoint" {
				mountpoint = label.GetValue()
			}
		}

		eventUUID, err := uuid.NewV4()
		if err != nil {
			return nil, err
		}

		createdAt := time.Now()
		event := repository.Event{
			EventUUID:   eventUUID.String(),
			AgentName:   agentName,
			ServiceName: _const.HARVESTMON_NODE_SERVICE_NAME,
			CommitID:    commitId,
			EventType:   _const.NODE_DISK_EVENT_TYPE,
			CreatedAt:   createdAt,
		}

		freeDiskStatuses = append(freeDiskStatuses, &repository.NodeFreeDiskStatus{
			Event:        event,
			CreatedAt:    createdAt,
			EventUUID:    eventUUID.String(),
			Mountpoint:   mountpoint,
			Device:       device,
			FreeDiskSize: freeDiskSize,
		})

	}

	return freeDiskStatuses, nil
}

func newNodeCPUSecondsTotal(agentName, commitId string, mfMap map[string]*dto.MetricFamily) ([]repository.StoreEntity, error) {
	var (
		storeEntities []repository.StoreEntity
		err           error
		m             *dto.MetricFamily
		exists        bool
	)
	if m, exists = mfMap[NODE_METRIC_CPU_SECONDS_TOTAL]; !exists {
		return nil, fmt.Errorf("failed to find node cpu seconds total")
	}

	for _, metric := range m.Metric {
		var (
			mode         string
			number       int
			secondsTotal uint64
		)

		secondsTotal = uint64(metric.GetCounter().GetValue())
		for _, label := range metric.GetLabel() {
			if label.GetName() == "mode" {
				mode = label.GetValue()
			} else if label.GetName() == "cpu" {
				number, err = strconv.Atoi(label.GetValue())
				if err != nil {
					return nil, err
				}
			}
		}

		eventUUID, err := uuid.NewV4()
		if err != nil {
			return nil, err
		}

		createdAt := time.Now()
		event := repository.Event{
			EventUUID:   eventUUID.String(),
			AgentName:   agentName,
			ServiceName: _const.HARVESTMON_NODE_SERVICE_NAME,
			CommitID:    commitId,
			EventType:   _const.NODE_CPU_EVENT_TYPE,
			CreatedAt:   createdAt,
		}

		storeEntities = append(storeEntities, &repository.NodeCpuSecondsTotal{
			Event:        event,
			CreatedAt:    createdAt,
			EventUUID:    eventUUID.String(),
			Mode:         mode,
			Number:       number,
			SecondsTotal: secondsTotal,
		})

	}

	return storeEntities, nil
}

func newNodeMemoryStatus(agentName, commitId string, mfMap map[string]*dto.MetricFamily) ([]repository.StoreEntity, error) {
	eventUUID, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}

	createdAt := time.Now()
	event := repository.Event{
		EventUUID:   eventUUID.String(),
		AgentName:   agentName,
		ServiceName: _const.HARVESTMON_NODE_SERVICE_NAME,
		CommitID:    commitId,
		EventType:   _const.NODE_MEMORY_EVENT_TYPE,
		CreatedAt:   createdAt,
	}

	var findValue = func(key string) (uint64, error) {
		if v, ok := mfMap[key]; ok {
			if len(v.GetMetric()) > 0 {
				return uint64(v.GetMetric()[0].GetGauge().GetValue()), nil
			}
		}
		return 0, errors.Errorf("%s not found", key)
	}

	var (
		memTotal      uint64
		memFree       uint64
		memBuffer     uint64
		memCached     uint64
		memSlab       uint64
		memPageTables uint64
		memSwapCached uint64
	)

	var keyVarMap = map[string]*uint64{
		NODE_MEM_TOTAL_BYTES_KEY:       &memTotal,
		NODE_MEM_FREE_BYTES_KEY:        &memFree,
		NODE_MEM_BUFFER_BYTES_KEY:      &memBuffer,
		NODE_MEM_CACHED_BYTES_KEY:      &memCached,
		NODE_MEM_SLAB_BYTES_KEY:        &memSlab,
		NODE_MEM_PAGETABLES_BYTES_KEY:  &memPageTables,
		NODE_MEM_SWAP_CACHED_BYTES_KEY: &memSwapCached,
	}

	for k, p := range keyVarMap {
		var v uint64
		if v, err = findValue(k); err != nil {
			return nil, err
		}
		*p = v
	}

	storeEntities := []repository.StoreEntity{
		&repository.NodeMemoryStatus{
			Event:     event,
			CreatedAt: createdAt,
			EventUUID: eventUUID.String(),

			Total:      memTotal,
			Free:       memFree,
			Buffer:     memBuffer,
			Cached:     memCached,
			Slab:       memSlab,
			PageTables: memPageTables,
			SwapCached: memSwapCached,
		},
	}
	return storeEntities, nil
}

func newNodeSystemdStatus(agentName, commitId string, mfMap map[string]*dto.MetricFamily) ([]repository.StoreEntity, error) {
	var (
		nodeSystemdStatuses []repository.StoreEntity
		m                   *dto.MetricFamily
		exists              bool
	)
	if m, exists = mfMap[NODE_SYSTEMD_UNIT_STATE_KEY]; !exists {
		return nil, errors.Errorf("%s not found", NODE_SYSTEMD_UNIT_STATE_KEY)
	}

	for _, metric := range m.Metric {
		if metric.GetGauge().GetValue() == 0 {
			continue
		}
		var (
			state     string
			daemoName string
		)
		for _, label := range metric.GetLabel() {
			if label.GetName() == "state" {
				state = label.GetValue()
			} else if label.GetName() == "name" {
				daemoName = label.GetValue()
			}
		}

		if daemoName == "" || state == "" {
			return nil, errors.Errorf("label not found. daemon: %v, state: %v", daemoName, state)
		}

		// it's too expensive to store all system daemon, and also their every state.
		// so, it'll filter only `active` state.
		if state != "active" {
			continue
		}

		eventUUID, err := uuid.NewV4()
		if err != nil {
			return nil, err
		}

		createdAt := time.Now()
		event := repository.Event{
			EventUUID:   eventUUID.String(),
			AgentName:   agentName,
			ServiceName: _const.HARVESTMON_NODE_SERVICE_NAME,
			CommitID:    commitId,
			EventType:   _const.NODE_SYSTEMD_EVENT_TYPE,
			CreatedAt:   createdAt,
		}

		nodeSystemdStatuses = append(nodeSystemdStatuses,
			&repository.NodeSystemdStatus{
				Event:     event,
				CreatedAt: createdAt,
				EventUUID: eventUUID.String(),

				SystemdName: daemoName,
				State:       state,
			})
	}

	return nodeSystemdStatuses, nil
}

func newNodeNetworkTotal(agentName, commitId string, mfMap map[string]*dto.MetricFamily) ([]repository.StoreEntity, error) {
	var (
		nodeNetworkTotals = make(map[string]repository.NodeNetworkTotal)
	)

	var findDeviceAndValue = func(metric *dto.Metric) (string, uint64) {
		var deviceName string
		for _, label := range metric.GetLabel() {
			if label.GetName() == "device" {
				deviceName = label.GetValue()
			}
		}
		if deviceName == "" {
		}

		return deviceName, uint64(metric.GetCounter().GetValue())
	}

	if mf, ok := mfMap[NODE_NETWORK_RECEIVE_TOTAL_KEY]; ok {
		for _, metric := range mf.GetMetric() {
			device, value := findDeviceAndValue(metric)
			if e, exists := nodeNetworkTotals[device]; exists {
				e.ReceiveTotal = value
				nodeNetworkTotals[device] = e
			} else {
				nodeNetworkTotals[device] = repository.NodeNetworkTotal{
					Device:       device,
					ReceiveTotal: value,
				}
			}
		}
	}

	if mf, ok := mfMap[NODE_NETWORK_TRANSMIT_TOTAL_KEY]; ok {
		for _, metric := range mf.GetMetric() {
			device, value := findDeviceAndValue(metric)
			if e, exists := nodeNetworkTotals[device]; exists {
				e.TransmitTotal = value
				nodeNetworkTotals[device] = e
			} else {
				nodeNetworkTotals[device] = repository.NodeNetworkTotal{
					Device:        device,
					TransmitTotal: value,
				}
			}
		}
	}

	var results []repository.StoreEntity
	for _, networkTotal := range nodeNetworkTotals {

		eventUUID, err := uuid.NewV4()
		if err != nil {
			return nil, err
		}

		createdAt := time.Now()
		event := repository.Event{
			EventUUID:   eventUUID.String(),
			AgentName:   agentName,
			ServiceName: _const.HARVESTMON_NODE_SERVICE_NAME,
			CommitID:    commitId,
			EventType:   _const.NODE_NETWORK_EVENT_TYPE,
			CreatedAt:   createdAt,
		}
		networkTotal.Event = event
		networkTotal.EventUUID = eventUUID.String()
		networkTotal.CreatedAt = createdAt
		results = append(results, &networkTotal)
	}

	return results, nil
}
