package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/b-harvest/Harvestmon/repository"
	"github.com/google/go-github/v43/github"
	"github.com/gorhill/cronexpr"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
	"sync"
	"time"
)

func InitializeViper() error {
	// Automatically bind environment variables
	viper.SetEnvPrefix("CHECKER") // All env vars should start with CHECKER_
	viper.AutomaticEnv()

	// Bind environment variables to specific configuration keys
	err := viper.BindEnv("commit_id", "CHECKER_COMMIT_ID")
	err = viper.BindEnv("database.user", "CHECKER_DATABASE_USER")
	err = viper.BindEnv("database.password", "CHECKER_DATABASE_PASSWORD")
	err = viper.BindEnv("database.host", "CHECKER_DATABASE_HOST")
	err = viper.BindEnv("database.port", "CHECKER_DATABASE_PORT")
	err = viper.BindEnv("database.dbName", "CHECKER_DATABASE_NAME")
	err = viper.BindEnv("database.maxIdleConns", "CHECKER_DATABASE_MAX_IDLE_CONNS")
	err = viper.BindEnv("database.maxOpenConns", "CHECKER_DATABASE_MAX_OPEN_CONNS")
	err = viper.BindEnv("database.connMaxLifeTime", "CHECKER_DATABASE_CONN_MAX_LIFE_TIME")
	err = viper.BindEnv("database.connMaxIdleTime", "CHECKER_DATABASE_CONN_MAX_IDLE_TIME")
	err = viper.BindEnv("database.awsRegion", "CHECKER_DATABASE_AWS_REGION")

	err = viper.BindEnv("read_database.user", "CHECKER_READ_DATABASE_USER")
	err = viper.BindEnv("read_database.password", "CHECKER_READ_DATABASE_PASSWORD")
	err = viper.BindEnv("read_database.host", "CHECKER_READ_DATABASE_HOST")
	err = viper.BindEnv("read_database.port", "CHECKER_READ_DATABASE_PORT")
	err = viper.BindEnv("read_database.dbName", "CHECKER_READ_DATABASE_NAME")
	err = viper.BindEnv("read_database.maxIdleConns", "CHECKER_READ_DATABASE_MAX_IDLE_CONNS")
	err = viper.BindEnv("read_database.maxOpenConns", "CHECKER_READ_DATABASE_MAX_OPEN_CONNS")
	err = viper.BindEnv("read_database.connMaxLifeTime", "CHECKER_READ_DATABASE_CONN_MAX_LIFE_TIME")
	err = viper.BindEnv("read_database.connMaxIdleTime", "CHECKER_READ_DATABASE_CONN_MAX_IDLE_TIME")
	err = viper.BindEnv("read_database.awsRegion", "CHECKER_READ_DATABASE_AWS_REGION")

	err = viper.BindEnv("github.credentials.username", "CHECKER_GITHUB_USERNAME")
	err = viper.BindEnv("github.credentials.token", "CHECKER_GITHUB_TOKEN")
	err = viper.BindEnv("github.branch", "CHECKER_GITHUB_BRANCH")
	err = viper.BindEnv("github.repository", "CHECKER_GITHUB_REPOSITORY")
	err = viper.BindEnv("github.nodeInfoPaths", "CHECKER_GITHUB_NODE_INFO_PATHS")
	err = viper.BindEnv("github.alarmerConfigPath", "CHECKER_GITHUB_ALARMER_PATH")

	if err != nil {
		return errors.New(fmt.Sprintf("error initializing viper: %v", err))
	}

	return nil
}

type Valid interface {
	checkValid() error
}

type CheckerConfig struct {
	logger *log.Entry

	// commitId specifies what version does it use
	CommitId string `mapstructure:"commit_id"`

	NodeInfos []NodeInfo `mapstructure:"node_infos"`

	// Check whole nodes' status task is very high computing resources used.
	// therefore, if not divide database into Read/Write, database may be hurt by high computation.
	//
	// if not set `read_database`, it'll be set using `database`(aka. WDatabase) configuration.
	//
	// Highly recommend to set both of database params.
	WDatabaseCfg Database `mapstructure:"database"`
	RDatabaseCfg Database `mapstructure:"read_database"`

	// Checker is designed to run on lambda, and it's pretty tedious, and complicated.
	//
	// through `github` config, you can set each nodes' configuration using github private repository simply.
	GithubFile GithubConfig `mapstructure:"github"`

	// AlarmerConfig is used to deal with whole alert.
	//
	// Because this parameter value will be inherited to child nodes' AlarmerConfig also,
	// highly recommended to set this param.
	//
	// if not set, nodes not set own AlarmerConfig can't notify current situation.
	AlarmerConfig AlarmerConfig `toml:"alarmer"`

	writeRepo *repository.Repository
	readRepo  *repository.Repository
}

func (c *CheckerConfig) checkValid() error {

	var (
		wg      sync.WaitGroup
		errChan = make(chan error)

		validNodeInfoMtx sync.Mutex
		validNodeInfos   []NodeInfo
	)
	for _, nodeInfo := range c.NodeInfos {
		wg.Add(1)
		go func(n NodeInfo) {
			defer wg.Done()
			if err := n.checkValid(); err != nil {
				errChan <- errors.New(fmt.Sprintf("invalid node %v: %v", n.Name, err))
				return
			}
			validNodeInfoMtx.Lock()
			validNodeInfos = append(validNodeInfos, n)
			validNodeInfoMtx.Unlock()
		}(nodeInfo)
	}
	go func() {
		for {
			select {
			case newErr := <-errChan:
				if newErr == nil {
					return
				}
				c.logger.Warning(newErr.Error())
			}
		}
	}()
	wg.Wait()
	errChan <- nil
	close(errChan)

	c.NodeInfos = validNodeInfos

	if c.CommitId == "" {
		return errors.New("CommitId is empty")
	}

	if c.logger == nil {
		return errors.New("logger is empty")
	}

	if c.readRepo == nil || c.writeRepo == nil {
		return errors.New("database[write, or read] is empty")
	}

	if len(c.NodeInfos) == 0 {
		return errors.New("NodeInfos is empty")
	}

	return nil
}

// NodeInfo defines target for watching
type NodeInfo struct {
	logger *log.Entry

	Name string `toml:"name"`

	// Alerts defines the types of alerts to send for this node.
	Alerts AlertConfigCollection `toml:"alerts"`

	// Alarmer in NodeInfo is used to deal with alert limited to its node.
	//
	// if not set, it'll be inherited from CheckerConfig.
	Alarmer *AlarmerConfig `toml:"alarmer"`

	strategyAlertStatuses map[StrategyTarget]*StrategyAlertStatus
}

func (n *NodeInfo) checkValid() error {
	if n.Name == "" {
		return errors.New("node name is empty")
	}

	if n.logger == nil {
		return errors.New("logger is empty")
	}

	return n.Alerts.checkValid()
}

type StrategyAlertStatus struct {
	alertMtx sync.Mutex

	// activeAlert uses alert event as key
	activeAlert map[alertEvent]time.Time

	// alarmCache uses combination of alarmer name and alertEvent as key to prevent double sending alarm.
	alarmCache map[alarmName]int64
}

// Snoozer option may be needed for node, or specific monitoring target(tendermint, ethereum, ...)
// therefore, Snoozer option is inherited from AlertConfigCollection to TendermintAlertConfig / EthereumAlertConfig.
// if Snoozer option is set for AlertConfigCollection, it'll apply SnoozeCron for node, and if it's set for specificAlertConfig, it'll be applied to AlertTarget only(tendermint, ethereum).
//
// if there is no SnoozeCron there, it'll return nil.
type Snoozer interface {
	isSnooze() bool
}

// SnoozeCron defines ignoring alert period.
// some nodes may have cron job affects to monitoringTarget's status. through this option, you can ignore alerts for specific period.
type SnoozeCron struct {
	StartCron string        `yaml:"startCron"`
	Duration  time.Duration `yaml:"duration"`
}

// AlertConfigCollection is collection of own node's alert configuration.
type AlertConfigCollection struct {
	Tendermint map[StrategyTarget]*TendermintAlertConfig `toml:"tendermint"`

	Ethereum map[StrategyTarget]*EthereumAlertConfig `toml:"ethereum"`

	// if set SnoozeCron, alert will not be occurred while specified period.
	SnoozeCrons []SnoozeCron `toml:"snoozes"`
}

func (c *AlertConfigCollection) checkValid() error {
	for _, t := range c.Tendermint {
		if err := t.checkValid(); err != nil {
			return err
		}
	}

	for _, e := range c.Ethereum {
		if err := e.checkValid(); err != nil {
			return err
		}
	}
	return nil
}

func (a *AlertConfigCollection) isSnooze() bool {
	for _, c := range a.SnoozeCrons {
		now := time.Now()
		beforeTime := time.Now().Add(-c.Duration)
		var nextTime time.Time
		for nextTime.Before(now) {
			beforeTime = beforeTime.Add(c.Duration)
			nextTime = cronexpr.MustParse(c.StartCron).Next(beforeTime)
		}

		cronStart := cronexpr.MustParse(c.StartCron).Next(beforeTime.Add(-c.Duration))

		if now.After(cronStart) && now.Before(cronStart.Add(c.Duration)) {
			return true
		}

	}
	return false
}

type AlertConfig interface {
	getAlertStrategies() []AlertStrategy
}

// AlertStrategy defines what alertLevel will this alert(e.g. tendermint, etheruem, ...) have.
//
// e.g.) if getAlertLevel() returns `high`, Alarmers have targetAlertLevels equal to `high` will be invoked.
type AlertStrategy interface {
	getName() string
	getAlertLevel() AlertLevel

	check(nodeName string) (alertEvent, string, bool)

	initialize(logger *log.Entry, repo *repository.Repository)
}

// Some instance may have multiple node.
// through specifying StrategyTarget, you can check every targets.
type StrategyTarget string

// TendermintAlertConfig collects configurations of each strategy have.
type TendermintAlertConfig struct {
	logger *log.Entry

	Height TendermintHeightStrategy `toml:"height"`
	Peer   TendermintPeerStrategy   `toml:"peer"`
	Commit TendermintCommitStrategy `toml:"commit"`

	// if set SnoozeCron, alert will not be occurred while specified period.
	SnoozeCrons []SnoozeCron `toml:"snoozes"`
}

func (a *TendermintAlertConfig) getAlertStrategies() []AlertStrategy {
	var strategies []AlertStrategy

	if a.Height.Enabled {
		strategies = append(strategies, &a.Height)
	}
	if a.Peer.Enabled {
		strategies = append(strategies, &a.Peer)
	}
	if a.Commit.Enabled {
		strategies = append(strategies, &a.Commit)
	}

	return strategies
}

func (t *TendermintAlertConfig) checkValid() error {
	for _, s := range t.getAlertStrategies() {
		if err := s.(Valid).checkValid(); err != nil {
			return err
		}
	}
	return nil
}

func (c *TendermintAlertConfig) isSnooze() bool {
	for _, c := range c.SnoozeCrons {
		now := time.Now()
		beforeTime := time.Now().Add(-c.Duration)
		var nextTime time.Time
		for nextTime.Before(now) {
			beforeTime = beforeTime.Add(c.Duration)
			nextTime = cronexpr.MustParse(c.StartCron).Next(beforeTime)
		}

		cronStart := cronexpr.MustParse(c.StartCron).Next(beforeTime.Add(-c.Duration))

		if now.After(cronStart) && now.Before(cronStart.Add(c.Duration)) {
			return true
		}

	}
	return false
}

// EthereumAlertConfig collects configurations of each alert strategy have.
type EthereumAlertConfig struct {
	logger *log.Entry

	Height EthereumHeightStrategy `toml:"height"`

	// EthereumBehindStrategy checks if it is falling behind to target(s)
	Behind EthereumBehindStrategy `toml:"behind"`

	// if set SnoozeCrons, alert will not be occurred while specified period.
	SnoozeCrons []SnoozeCron `toml:"snoozes"`
}

func (a *EthereumAlertConfig) getAlertStrategies() []AlertStrategy {
	var strategies []AlertStrategy
	if a.Height.Enabled {
		strategies = append(strategies, &a.Height)
	}
	if a.Behind.Enabled {
		strategies = append(strategies, &a.Behind)
	}

	return strategies
}

func (e *EthereumAlertConfig) checkValid() error {
	for _, s := range e.getAlertStrategies() {
		if err := s.(Valid).checkValid(); err != nil {
			return err
		}
	}
	return nil
}

func (s *EthereumAlertConfig) isSnooze() bool {
	for _, c := range s.SnoozeCrons {
		now := time.Now()
		beforeTime := time.Now().Add(-c.Duration)
		var nextTime time.Time
		for nextTime.Before(now) {
			beforeTime = beforeTime.Add(c.Duration)
			nextTime = cronexpr.MustParse(c.StartCron).Next(beforeTime)
		}

		cronStart := cronexpr.MustParse(c.StartCron).Next(beforeTime.Add(-c.Duration))

		if now.After(cronStart) && now.Before(cronStart.Add(c.Duration)) {
			return true
		}

	}
	return false
}

// GithubConfig defines the path to fetch configurations for nodeInfo
type GithubConfig struct {
	GithubCredentials struct {
		Username string `toml:"username"`
		// if not set, target repository will be considered publicly.
		Token string `toml:"token"`
	} `mapstructure:"credentials"`

	// Branch indicates what branch will be used to read config files.
	//
	// if not set, defaults to `main`.
	Branch string `toml:"branch"`

	// Repository indicates only repository name.
	//
	// e.g. if you want to use github.com/example/example-config, you have to set repository to `example-config`.
	// 		and also set Username to `exapmle`.
	Repository string `toml:"repository"`

	// NodeInfoPaths indicates what directory/files should be read.
	//
	// if not set, every files in remote repository will be read.
	NodeInfoPaths []string `mapstructure:"nodeInfoPaths"`

	// Because it's designed to run on lambda, there may be some difficult to set config if you have to put config file on the same place with checker program.
	// therefore, it allows you to set alarmerConfig remotely.
	AlarmerConfigPath string `toml:"alarmerConfigPath"`

	client *github.Client

	lastCommit *github.RepositoryCommit
}

func (g *GithubConfig) isChanged() bool {
	if g.client == nil {
		// Setup authentication
		ctx := context.Background()
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: g.GithubCredentials.Token},
		)
		tc := oauth2.NewClient(ctx, ts)
		gc := github.NewClient(tc)
		g.client = gc
	}

	commits, _, err := g.client.Repositories.ListCommits(context.Background(), g.GithubCredentials.Username, g.Repository, nil)
	if err != nil {
		return false
	}

	if len(commits) > 0 {
		if g.lastCommit == nil {
			g.lastCommit = commits[0]
		}
		isChanged := commits[0].SHA != g.lastCommit.SHA
		g.lastCommit = commits[0]

		return isChanged
	}
	return false
}

func (g *GithubConfig) getFilesBytes(path string) [][]byte {
	if g.client == nil {
		// Setup authentication
		ctx := context.Background()
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: g.GithubCredentials.Token},
		)
		tc := oauth2.NewClient(ctx, ts)
		gc := github.NewClient(tc)
		g.client = gc
	}

	fileContent, dirContent, _, err := g.client.Repositories.GetContents(context.Background(), g.GithubCredentials.Username, g.Repository, path, nil)
	if err != nil {
		return nil
	}

	var (
		result [][]byte
		mtx    sync.Mutex
	)
	if fileContent == nil {
		var wg sync.WaitGroup
		for _, dc := range dirContent {
			wg.Add(1)
			go func() {
				defer wg.Done()
				res := g.getFilesBytes(*dc.Path)
				mtx.Lock()
				result = append(result, res...)
				mtx.Unlock()
			}()
		}
		wg.Wait()
	} else {
		// Decode the content of the file from base64
		content, err := fileContent.GetContent()
		if err != nil {
			return nil
		}
		mtx.Lock()
		result = append(result, []byte(content))
		mtx.Unlock()
	}

	return result
}
