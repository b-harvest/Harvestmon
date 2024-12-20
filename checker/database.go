package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/rds/auth"
	"github.com/b-harvest/Harvestmon/repository"
	"github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
	gorm_mysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"strconv"
	"time"
)

type Database struct {
	User     string `toml:"user"`
	Password string `toml:"password"`
	Host     string `toml:"host"`
	Port     int    `toml:"port"`
	DbName   string `toml:"dbName"`

	// if set, checker will try to connect rds.
	AwsRegion string `toml:"awsRegion"`

	maxIdleConns    int           `toml:"maxIdleConns"`
	maxOpenConns    int           `toml:"maxOpenConns"`
	connMaxLifeTime time.Duration `toml:"connMaxLifeTime"`
	connMaxIdleTime time.Duration `toml:"connMaxIdleTime"`
}

func GetDatabase(dbConfig Database) (*sql.DB, error) {
	var (
		db  *sql.DB
		err error
	)
	if dbConfig.AwsRegion == "" {
		db, err = getDBConnection(dbConfig)
		if err != nil {
			return nil, err
		}
	} else {
		db, err = getRDSConnection(dbConfig)
		if err != nil {
			return nil, err
		}
	}

	db.SetMaxIdleConns(dbConfig.maxIdleConns)
	db.SetMaxOpenConns(dbConfig.maxOpenConns)
	db.SetConnMaxLifetime(dbConfig.connMaxIdleTime)
	db.SetConnMaxIdleTime(dbConfig.connMaxIdleTime)

	return db, nil
}

func getDBConnection(dbConfig Database) (*sql.DB, error) {

	mysqlDB := mysql.Config{
		User:                 dbConfig.User,
		Passwd:               dbConfig.Password,
		Net:                  "tcp",
		Addr:                 fmt.Sprintf("%s:%s", dbConfig.Host, strconv.Itoa(dbConfig.Port)),
		Collation:            "utf8mb4_general_ci",
		ParseTime:            true,
		Loc:                  time.UTC,
		MaxAllowedPacket:     4 << 20.,
		AllowNativePasswords: true,
		CheckConnLiveness:    true,
		DBName:               dbConfig.DbName,
	}
	connector, err := mysql.NewConnector(&mysqlDB)
	if err != nil {
		return nil, err
	}

	db := sql.OpenDB(connector)
	return db, nil
}

func getRDSConnection(dbConfig Database) (*sql.DB, error) {
	var dbEndpoint = fmt.Sprintf("%s:%d", dbConfig.Host, dbConfig.Port)
	var region = dbConfig.AwsRegion
	if region == "" {
		region = "us-east-1"
	}

	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		panic("configuration error: " + err.Error())
	}

	authenticationToken, err := auth.BuildAuthToken(
		context.Background(), dbEndpoint, region, dbConfig.User, cfg.Credentials)
	if err != nil {
		panic("failed to create authentication token: " + err.Error())
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?tls=true&allowCleartextPasswords=true&parseTime=True",
		dbConfig.User, authenticationToken, dbEndpoint, dbConfig.DbName,
	)

	db, err := sql.Open("mysql", dsn)
	return db, err
}

func (cc *CheckerConfig) getRepository(db *sql.DB) (*repository.Repository, error) {
	gormDB, err := gorm.Open(gorm_mysql.New(gorm_mysql.Config{Conn: db}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent), CreateBatchSize: 100})
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to database")
	}

	return &repository.Repository{DB: *gormDB, CommitId: cc.CommitId}, nil
}
