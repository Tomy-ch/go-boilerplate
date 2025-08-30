// Package migrate は、データベースのマイグレーションに関する機能を提供します。
package migrate

import (
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

	return migrate.New("file://"+migrateFilePlace, cfg.DatabaseDSN())
}
