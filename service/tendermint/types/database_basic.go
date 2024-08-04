//go:build !rds

package types

import (
	"database/sql"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"strconv"
	"time"
)

func (r *MonitorClient) GetDatabase() *sql.DB {
	return r.db
}

func getDatabase(mConfig *MonitorConfig) *sql.DB {
	cfg := mysql.Config{
		User:                 mConfig.Database.User,
		Passwd:               mConfig.Database.Password,
		Net:                  "tcp",
		Addr:                 fmt.Sprintf("%s:%s", mConfig.Database.Host, strconv.Itoa(mConfig.Database.Port)),
		Collation:            "utf8mb4_general_ci",
		Loc:                  time.UTC,
		MaxAllowedPacket:     4 << 20.,
		AllowNativePasswords: true,
		CheckConnLiveness:    true,
		DBName:               mConfig.Database.DbName,
	}
	connector, err := mysql.NewConnector(&cfg)
	if err != nil {
		panic(err)
	}
	db := sql.OpenDB(connector)

	return db
}
