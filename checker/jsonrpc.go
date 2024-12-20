package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type JsonRPC struct {
	httpClient *http.Client
}

type JsonRPCResponse struct {
	Jsonrpc string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result"`
	Error   *RPCErr         `json:"error"`
	ID      int             `json:"id"`
}

type RPCErr struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (j JsonRPC) Call(rpcUrl, method string, params any) (*JsonRPCResponse, error) {
	reqBody, err := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
		"id":      1,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(
		"POST",
		rpcUrl,
		bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := j.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HTTP response status: %s", resp.Status)
	}

	c, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var res JsonRPCResponse
	err = json.Unmarshal(c, &res)
	if err != nil {
		return nil, err
	}

	return &res, nil
}
