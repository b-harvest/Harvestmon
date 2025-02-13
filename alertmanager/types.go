package main

import (
	"context"
	"fmt"
	"github.com/b-harvest/Harvestmon/repository"
	"github.com/google/go-github/v43/github"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
	"sync"
	"time"
)

func InitializeViper() error {
	// Automatically bind environment variables
	viper.SetEnvPrefix("ALERT") // All env vars should start with ALERT_
	viper.AutomaticEnv()

	// Bind environment variables to specific configuration keys
	err := viper.BindEnv("commit_id", "ALERT_COMMIT_ID")
	err = viper.BindEnv("database.user", "ALERT_DATABASE_USER")
	err = viper.BindEnv("database.password", "ALERT_DATABASE_PASSWORD")
	err = viper.BindEnv("database.host", "ALERT_DATABASE_HOST")
	err = viper.BindEnv("database.port", "ALERT_DATABASE_PORT")
	err = viper.BindEnv("database.dbName", "ALERT_DATABASE_NAME")
	err = viper.BindEnv("database.maxIdleConns", "ALERT_DATABASE_MAX_IDLE_CONNS")
	err = viper.BindEnv("database.maxOpenConns", "ALERT_DATABASE_MAX_OPEN_CONNS")
	err = viper.BindEnv("database.connMaxLifeTime", "ALERT_DATABASE_CONN_MAX_LIFE_TIME")
	err = viper.BindEnv("database.connMaxIdleTime", "ALERT_DATABASE_CONN_MAX_IDLE_TIME")
	err = viper.BindEnv("database.awsRegion", "ALERT_DATABASE_AWS_REGION")

	err = viper.BindEnv("read_database.user", "ALERT_READ_DATABASE_USER")
	err = viper.BindEnv("read_database.password", "ALERT_READ_DATABASE_PASSWORD")
	err = viper.BindEnv("read_database.host", "ALERT_READ_DATABASE_HOST")
	err = viper.BindEnv("read_database.port", "ALERT_READ_DATABASE_PORT")
	err = viper.BindEnv("read_database.dbName", "ALERT_READ_DATABASE_NAME")
	err = viper.BindEnv("read_database.maxIdleConns", "ALERT_READ_DATABASE_MAX_IDLE_CONNS")
	err = viper.BindEnv("read_database.maxOpenConns", "ALERT_READ_DATABASE_MAX_OPEN_CONNS")
	err = viper.BindEnv("read_database.connMaxLifeTime", "ALERT_READ_DATABASE_CONN_MAX_LIFE_TIME")
	err = viper.BindEnv("read_database.connMaxIdleTime", "ALERT_READ_DATABASE_CONN_MAX_IDLE_TIME")
	err = viper.BindEnv("read_database.awsRegion", "ALERT_READ_DATABASE_AWS_REGION")

	err = viper.BindEnv("github.credentials.username", "ALERT_GITHUB_USERNAME")
	err = viper.BindEnv("github.credentials.token", "ALERT_GITHUB_TOKEN")
	err = viper.BindEnv("github.branch", "ALERT_GITHUB_BRANCH")
	err = viper.BindEnv("github.repository", "ALERT_GITHUB_REPOSITORY")
	err = viper.BindEnv("github.path", "ALERT_GITHUB_PATH")

	if err != nil {
		return errors.New(fmt.Sprintf("error initializing viper: %v", err))
	}

	return nil
}

type InstanceName string
type TargetName string

type AlertManager struct {
	logger *log.Entry

	mu *sync.RWMutex

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
	GithubFile *GithubConfig `mapstructure:"github"`

	// AlarmerConfig is used to deal with whole alert.
	//
	// Because this parameter value will be inherited to child nodes' AlarmerConfig also,
	// highly recommended to set this param.
	//
	// if not set, nodes not set own AlarmerConfig can't notify current situation.
	AlarmerConfig *AlarmerConfig `toml:"alarmer"`

	reader *repository.Repository
	writer *repository.Repository

	record map[InstanceName]map[TargetName]*Record
}

func (am *AlertManager) checkValid() error {
	if am == nil {
		return errors.New("nil AlertManager")
	}
	if am.AlarmerConfig == nil {
		return errors.New("empty AlarmerConfig")
	}

	return nil
}

func NewAlertManager(logger *log.Entry) *AlertManager {
	return &AlertManager{
		logger: logger,
		mu:     &sync.RWMutex{},
	}
}

type Record struct {
	mtx *sync.Mutex

	// activeAlert uses alert event as key
	activeAlert map[alertEvent]time.Time

	// alarmCache uses combination of alarmer name and alertEvent as key to prevent double sending alarm.
	alarmCache map[alarmName]int64
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

	// Path indicates what directory/files should be read.
	Path string `toml:"path"`

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

	var opts *github.RepositoryContentGetOptions
	if g.Branch != "" {
		opts = &github.RepositoryContentGetOptions{Ref: g.Branch}
	}
	fileContent, dirContent, _, err := g.client.Repositories.GetContents(context.Background(), g.GithubCredentials.Username, g.Repository, path, opts)
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
