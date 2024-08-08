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

func (r *CheckerClient) GetDatabase() *gorm.DB {
	gormDB, err := gorm.Open(gorm_mysql.New(gorm_mysql.Config{Conn: r.DB}))
	if err != nil {
		panic(err)
	}
	return gormDB
}

func getDatabase(checkerConfig *CheckerConfig) *sql.DB {
	cfg := mysql.Config{
		User:                 checkerConfig.Database.User,
		Passwd:               checkerConfig.Database.Password,
		Net:                  "tcp",
		Addr:                 fmt.Sprintf("%s:%s", checkerConfig.Database.Host, strconv.Itoa(checkerConfig.Database.Port)),
		Collation:            "utf8mb4_general_ci",
		ParseTime:            true,
		Loc:                  time.UTC,
		MaxAllowedPacket:     4 << 20.,
		AllowNativePasswords: true,
		CheckConnLiveness:    true,
		DBName:               checkerConfig.Database.DbName,
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
