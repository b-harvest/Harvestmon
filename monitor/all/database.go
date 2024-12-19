package main

import (
	"database/sql"
	"fmt"
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

	maxIdleConns    int           `toml:"maxIdleConns"`
	maxOpenConns    int           `toml:"maxOpenConns"`
	connMaxLifeTime time.Duration `toml:"connMaxLifeTime"`
	connMaxIdleTime time.Duration `toml:"connMaxIdleTime"`

	DbBatchSize int `mapstructure:"dbBatchSize"`
}

func (d *Database) Validate() error {
	if d.User == "" {
		return errors.New("user is required")
	}
	if d.Password == "" {
		return errors.New("password is required")
	}
	if d.Host == "" {
		return errors.New("host is required")
	}
	if d.Port == 0 {
		return errors.New("port is required")
	}
	if d.DbName == "" {
		return errors.New("dbName is required")
	}
	return nil
}

func GetDatabase(dbConfig *Database) (*sql.DB, error) {
	var (
		db  *sql.DB
		err error
	)
	db, err = getDBConnection(dbConfig)
	if err != nil {
		return nil, err
	}

	db.SetMaxIdleConns(dbConfig.maxIdleConns)
	db.SetMaxOpenConns(dbConfig.maxOpenConns)
	db.SetConnMaxLifetime(dbConfig.connMaxIdleTime)
	db.SetConnMaxIdleTime(dbConfig.connMaxIdleTime)

	return db, nil
}

func getDBConnection(dbConfig *Database) (*sql.DB, error) {

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

func (c *Config) getRepository() (*repository.BaseRepository, error) {
	gormDB, err := gorm.Open(gorm_mysql.New(gorm_mysql.Config{Conn: c.db}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent), CreateBatchSize: 100})
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to database")
	}

	return &repository.BaseRepository{DB: *gormDB, CommitId: c.CommitId}, nil
}
