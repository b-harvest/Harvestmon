package types

import (
	"encoding/json"
	"time"
)

type Monitor interface {
	Run(c *MonitorConfig, rpcClient *MonitorClient)
}

type Func struct {
	MonitorFunc `yaml:"name"`
	Interval    *time.Duration `yaml:"interval"`
}

type MonitorFunc func(c *MonitorConfig, rpcClient *MonitorClient)

func (f Func) Run(c *MonitorConfig, rpcClient *MonitorClient) {
	f.MonitorFunc(c, rpcClient)
}

type RPCResponse struct {
	Jsonrpc string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result"`
	Error   *RPCErr         `json:"error"`
	ID      int             `json:"id"`
}

type RPCErr struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
