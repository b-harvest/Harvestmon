//go:build rds

package types

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/rds/auth"
)

func GetDatabase(database *Database) *sql.DB {
	var dbName = database.DbName
	var dbUser = database.User
	var dbHost = database.Host
	var dbEndpoint = fmt.Sprintf("%s:%d", dbHost, database.Port)
	var region = database.AwsRegion
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
