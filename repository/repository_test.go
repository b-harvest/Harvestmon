package repository

import (
	"database/sql"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	gorm_mysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"testing"
	"time"
)

func Test(t *testing.T) {

	cfg := mysql.Config{
		User:                 "root",
		Passwd:               "accounting-mysql",
		Net:                  "tcp",
		Addr:                 "127.0.0.1:33306",
		Collation:            "utf8mb4_general_ci",
		Loc:                  time.UTC,
		ParseTime:            true,
		MaxAllowedPacket:     4 << 20.,
		AllowNativePasswords: true,
		CheckConnLiveness:    true,
		DBName:               "harvestmon",
	}
	connector, _ := mysql.NewConnector(&cfg)
	db := sql.OpenDB(connector)
	db.SetMaxIdleConns(5)
	db.SetMaxOpenConns(5)

	gormDB, _ := gorm.Open(gorm_mysql.New(gorm_mysql.Config{Conn: db}))

	t.Run("select raw test", func(t *testing.T) {
		commitRepository := CommitRepository{EventRepository{DB: *gormDB, CommitId: "test-commit-id"}}

		result, err := commitRepository.FindValidatorAddressesWithAgents("000001E443FD237E4B616E2FA69DF4EE3D49A94F", 50)
		assert.NoError(t, err)

		fmt.Println(result)

	})

}
