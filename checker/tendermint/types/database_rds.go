//go:build rds

package types

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/rds/auth"
)

func (r *MonitorClient) GetDatabase() *sql.DB {
	return r.db
}

func getDatabase(mConfig *MonitorConfig) *sql.DB {
	var dbName = mConfig.Database.dbName
	var dbUser = mConfig.Database.dbUser
	var dbHost = mConfig.Database.host
	var dbPort = mConfig.Database.port
	var dbEndpoint = fmt.Sprintf("%s:%s?parseTime=true", dbHost, strconv.Itoa(mConfig.Database.port))
	var region = mConfig.Database.awsRegion
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

	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?tls=true&allowCleartextPasswords=true",
		dbUser, authenticationToken, dbEndpoint, dbName,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}

	return db
}
