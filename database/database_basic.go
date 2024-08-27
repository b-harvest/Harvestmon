//go:build !rds

package types

import (
	"database/sql"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"gopkg.in/yaml.v3"
	"os"
	"strconv"
	"time"
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

	cfg := mysql.Config{
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
	connector, err := mysql.NewConnector(&cfg)
	if err != nil {
		panic(err)
	}
	db := sql.OpenDB(connector)
	db.SetMaxIdleConns(5)
	db.SetMaxOpenConns(5)
	db.SetConnMaxLifetime(0)
	db.SetConnMaxIdleTime(1 * time.Minute)

	return db, nil
}
