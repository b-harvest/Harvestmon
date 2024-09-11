//go:build rds

package types

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/rds/auth"
	"gopkg.in/yaml.v3"
	"os"
	"strconv"
	"time"
)

func GetDatabase(defaultFilePath, envPrefix string) (*sql.DB, error) {

	dbConfig := new(Database)

	var err error
	configBytes, err := os.ReadFile(defaultFilePath)
	if err != nil {
		err = nil
	} else {
		err = yaml.Unmarshal(configBytes, &dbConfig)
		if err != nil {
			return nil, err
		}
	}

	if dbConfig.User == "" {
		dbConfig.User = os.Getenv(envPrefix + EnvDBUser)
	}
	if dbConfig.Password == "" {
		dbConfig.Password = os.Getenv(envPrefix + EnvDBPassword)
	}
	if dbConfig.Host == "" {
		dbConfig.Host = os.Getenv(envPrefix + EnvDBHost)
	}
	if dbConfig.Port == 0 {
		port, _ := strconv.Atoi(os.Getenv(envPrefix + EnvDBPort))
		dbConfig.Port = port
	}
	if dbConfig.DbName == "" {
		dbConfig.DbName = os.Getenv(envPrefix + EnvDBName)
	}
	if dbConfig.AwsRegion == "" {
		dbConfig.AwsRegion = os.Getenv(envPrefix + EnvDBAwsRegion)
	}

	var dbName = dbConfig.DbName
	var dbUser = dbConfig.User
	var dbHost = dbConfig.Host
	var dbEndpoint = fmt.Sprintf("%s:%d", dbHost, dbConfig.Port)
	var region = dbConfig.AwsRegion
	if region == "" {
		region = "us-east-1"
	}

	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		panic("configuration error: " + err.Error())
	}

	authenticationToken, err := auth.BuildAuthToken(
		context.Background(), dbEndpoint, region, dbUser, cfg.Credentials)
	if err != nil {
		panic("failed to create authentication token: " + err.Error())
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?tls=true&allowCleartextPasswords=true&parseTime=True",
		dbUser, authenticationToken, dbEndpoint, dbName,
	)

	db, err := sql.Open("mysql", dsn)
	var (
		maxIdleConns    int
		maxOpenConns    int
		connMaxLifeTime time.Duration
		connMaxIdleTime time.Duration
	)

	if maxIdleConnsStr := os.Getenv(EnvMaxIdleConns); maxIdleConnsStr == "" {
		maxIdleConns = 5
	} else {
		maxIdleConns, err = strconv.Atoi(maxIdleConnsStr)
		if err != nil {
			maxIdleConns = 5
		}
	}

	if maxOpenConnsStr := os.Getenv(EnvMaxOpenConns); maxOpenConnsStr == "" {
		maxOpenConns = 5
	} else {
		maxOpenConns, err = strconv.Atoi(maxOpenConnsStr)
		if err != nil {
			maxOpenConns = 5
		}
	}

	if connMaxLifeTimeStr := os.Getenv(EnvConnMaxLifeTime); connMaxLifeTimeStr == "" {
		connMaxLifeTime = 0
	} else {
		connMaxLifeTime, err = time.ParseDuration(connMaxLifeTimeStr)
		if err != nil {
			connMaxLifeTime = 0
		}
	}

	if connMaxIdleTimeStr := os.Getenv(EnvConnMaxIdleTime); connMaxIdleTimeStr == "" {
		connMaxIdleTime = 1 * time.Minute
	} else {
		connMaxIdleTime, err = time.ParseDuration(connMaxIdleTimeStr)
		if err != nil {
			connMaxIdleTime = 1 * time.Minute
		}
	}

	db.SetMaxIdleConns(maxIdleConns)
	db.SetMaxOpenConns(maxOpenConns)
	db.SetConnMaxLifetime(connMaxLifeTime)
	db.SetConnMaxIdleTime(connMaxIdleTime)
	if err != nil {
		panic(err)
	}

	return db, err
}
