package types

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/b-harvest/Harvestmon/log"
	"io"
	"net/http"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	statusEndpoint  = "/status"
	netInfoEndpoint = "/net_info"
	commitEndpoint  = "/commit"
)

type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type MonitorClient struct {
	httpClient   HttpClient
	hostWithPort string
	timeout      time.Duration
	retries      int
	DB           *sql.DB
}

func NewMonitorClient(cfg *MonitorConfig, httpClient HttpClient) *MonitorClient {
	hostWithPort := fmt.Sprintf("%s:%s", cfg.Agent.Host, strconv.Itoa(cfg.Agent.Port))

	rpcClient := MonitorClient{
		httpClient:   httpClient,
		hostWithPort: hostWithPort,
		timeout:      cfg.Agent.Timeout,
		retries:      3,
		DB:           getDatabase(cfg),
	}
	return &rpcClient
}

func (r *MonitorClient) GetCometBFTStatus() (*ResultStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()
	req, err := requestGet(ctx, r.getAddress(statusEndpoint))
	if err != nil {
		funcName := runtime.FuncForPC(reflect.ValueOf(r.GetCometBFTStatus).Pointer()).Name()
		return nil, errors.New("Could not fetch rpc status. functionName: " + funcName + ", err: " + err.Error())
	}

	var (
		body         []byte
		statusResult CometBFTStatusResult
	)
	body, err = request(r.httpClient, req, r.retries)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &statusResult)
	if err != nil {
		return nil, errors.New("Json marshaling error: " + err.Error())
	}
	return &statusResult.Result, nil
}

func (r *MonitorClient) GetNetInfo() (*CometBFTNetInfoResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()
	req, err := requestGet(ctx, r.getAddress(netInfoEndpoint))
	if err != nil {
		funcName := runtime.FuncForPC(reflect.ValueOf(r.GetCometBFTStatus).Pointer()).Name()
		return nil, errors.New("Could not fetch rpc status. functionName: " + funcName + ", err: " + err.Error())
	}

	var (
		body         []byte
		resultStatus CometBFTNetInfoResult
	)
	body, err = request(r.httpClient, req, r.retries)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &resultStatus)
	if err != nil {
		return nil, err
	}

	return &resultStatus, nil
}

func (r *MonitorClient) GetCommit() (*CometBFTCommitResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()
	req, err := requestGet(ctx, r.getAddress(commitEndpoint))
	if err != nil {
		funcName := runtime.FuncForPC(reflect.ValueOf(r.GetCometBFTStatus).Pointer()).Name()
		return nil, errors.New("Could not fetch rpc status. functionName: " + funcName + ", err: " + err.Error())
	}

	var (
		body         []byte
		resultStatus CometBFTCommitResult
	)
	body, err = request(r.httpClient, req, r.retries)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &resultStatus)
	if err != nil {
		return nil, err
	}

	return &resultStatus, nil
}

func (r *MonitorClient) GetCommitWithHeight(height uint64) (*CometBFTCommitResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()
	req, err := requestGet(ctx, fmt.Sprintf("%s?height=%d", r.getAddress(commitEndpoint), height))
	if err != nil {
		funcName := runtime.FuncForPC(reflect.ValueOf(r.GetCometBFTStatus).Pointer()).Name()
		return nil, errors.New("Could not fetch rpc status. functionName: " + funcName + ", err: " + err.Error())
	}

	var (
		body         []byte
		resultStatus CometBFTCommitResult
	)
	body, err = request(r.httpClient, req, r.retries)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &resultStatus)
	if err != nil {
		return nil, err
	}

	return &resultStatus, nil
}

func requestGet(ctx context.Context, address string) (*http.Request, error) {
	return http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
}

func request(c HttpClient, request *http.Request, retries int) ([]byte, error) {
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

func (r *MonitorClient) getAddress(endpoint string) string {
	hostName := r.hostWithPort
	if strings.Contains(hostName, "http") {
		return fmt.Sprintf("%s%s", hostName, endpoint)
	} else if strings.Contains(hostName, "443") {
		return fmt.Sprintf("https://%s%s", hostName, endpoint)
	} else {
		return fmt.Sprintf("http://%s%s", hostName, endpoint)
	}

}
