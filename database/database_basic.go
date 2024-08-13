//go:build !rds

package types

import (
	"database/sql"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"strconv"
	"time"
)

func GetDatabase(database *Database) *sql.DB {
	cfg := mysql.Config{
		User:                 database.User,
		Passwd:               database.Password,
		Net:                  "tcp",
		Addr:                 fmt.Sprintf("%s:%s", database.Host, strconv.Itoa(database.Port)),
		Collation:            "utf8mb4_general_ci",
		ParseTime:            true,
		Loc:                  time.UTC,
		MaxAllowedPacket:     4 << 20.,
		AllowNativePasswords: true,
		CheckConnLiveness:    true,
		DBName:               database.DbName,
	}
	connector, err := mysql.NewConnector(&cfg)
	if err != nil {
		panic(err)
	}
	db := sql.OpenDB(connector)
	db.SetMaxIdleConns(5)
	db.SetMaxOpenConns(5)

	return db
}
