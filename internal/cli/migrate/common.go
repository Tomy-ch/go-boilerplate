// Package migrate は、データベースのマイグレーションに関する機能を提供します。
package migrate

import (
	"os"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/rdb/driver"

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
	// まず通常の設定を読み込み、必要に応じて対象 DB 名だけ CLI 引数で差し替えます。
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
	dbCfg := config.NewDatabaseConfig(cfg)
	osCfg := config.NewOperationSystemConfig(cfg)

	// ファイルシステム上の migration 群と、実行先 DB の DSN を結び付けて migrate を生成します。
	return migrate.New("file://"+migrateFilePlace, driver.DSNWithTimeZoneString(dbCfg, osCfg))
}
