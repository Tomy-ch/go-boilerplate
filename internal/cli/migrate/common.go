// Package migrate は、データベースのマイグレーションに関する機能を提供します。
package migrate

import (
	"fmt"
	"net/url"
	"os"

	"boilerplate-go/internal/config"

	"github.com/golang-migrate/migrate/v4"
)

const (
	// migrateFilePlace は、マイグレーションファイルの場所を定義します。
	migrateFilePlace = "database/migrations"
)

var (
	// マイグレーションのターゲットバージョン
	targetVersion int
	// マイグレーションのターゲットデータベース
	targetDatabase string
)

// buildMigrateInstance は、マイグレーションインスタンスを生成します。
func buildMigrateInstance(tgtDB string) (*migrate.Migrate, error) {
	err := config.Load()
	if err != nil {
		return nil, err
	}
	if tgtDB != "" {
		err = os.Setenv("DB_NAME", tgtDB)
		if err != nil {
			return nil, err
		}
	}
	cfg, err := config.New()
	if err != nil {
		return nil, err
	}
	fmt.Println("env:", os.Getenv("ENV"))
	fmt.Println("AppENV:", cfg.AppEnv())
	fmt.Println("cfg.DatabaseDSN():", cfg.DatabaseDSN())
	fmt.Println("database.user:", cfg.DatabaseUser())
	fmt.Println("database.password:", cfg.DatabasePassword())
	fmt.Println("database.host:", cfg.DatabaseHost())
	fmt.Println("database.port:", cfg.DatabasePort())
	fmt.Println("database.name:", cfg.DatabaseName())
	fmt.Println("database.sslMode:", cfg.DatabaseSSLMode())
	fmt.Println("database.timezone:", url.QueryEscape(cfg.OSTimeZone()))

	return migrate.New("file://"+migrateFilePlace, cfg.DatabaseDSN())
}
