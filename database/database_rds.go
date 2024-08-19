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
)

func GetDatabase(defaultFilePath string) (*sql.DB, error) {

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
		dbConfig.User = os.Getenv(EnvDBUser)
	}
	if dbConfig.Password == "" {
		dbConfig.Password = os.Getenv(EnvDBPassword)
	}
	if dbConfig.Host == "" {
		dbConfig.Host = os.Getenv(EnvDBHost)
	}
	if dbConfig.Port == 0 {
		port, _ := strconv.Atoi(os.Getenv(EnvDBPort))
		dbConfig.Port = port
	}
	if dbConfig.DbName == "" {
		dbConfig.DbName = os.Getenv(EnvDBName)
	}
	if dbConfig.AwsRegion == "" {
		dbConfig.AwsRegion = os.Getenv(EnvDBAwsRegion)
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
	if err != nil {
		panic(err)
	}

	return db, err
}
