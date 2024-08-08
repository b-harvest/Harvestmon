//go:build !rds

package types

import (
	"database/sql"
	"fmt"
	"github.com/go-sql-driver/mysql"
	gorm_mysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"strconv"
	"time"
)

func (r *MonitorClient) GetDatabase() *gorm.DB {
	gormDB, err := gorm.Open(gorm_mysql.New(gorm_mysql.Config{Conn: r.DB}))
	if err != nil {
		panic(err)
	}
	return gormDB
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
