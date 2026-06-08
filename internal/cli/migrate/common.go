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

//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE -package=mock_$GOPACKAGE

// migrator は、golang-migrate の操作を抽象化し、テストでモック差し替え可能にします。
type migrator interface {
	Up() error
	Down() error
	Steps(n int) error
	Version() (version uint, dirty bool, err error)
	Force(version int) error
}

// migratorFactory は、対象 DB 名から migrator を生成する関数型です。
type migratorFactory func(database string) (migrator, error)

// buildMigrateInstance は、マイグレーションインスタンスを生成します。
func buildMigrateInstance(database string) (migrator, error) {
	// まず通常の設定を読み込み、必要に応じて対象 DB 名だけ CLI 引数で差し替えます。
	if err := config.Load(); err != nil {
		return nil, err
	}
	if database != "" {
		// config.New() が読み取る間だけ DB_NAME を差し替え、読み取り後は元値へ復元して冪等性を保ちます。
		restore := overrideEnv("DB_NAME", database)
		defer restore()
	}
	cfg, err := config.New()
	if err != nil {
		return nil, err
	}
	dbCfg := config.NewDatabaseConfig(cfg)
	osCfg := config.NewOperationSystemConfig(cfg)

	// ファイルシステム上の migration 群と、実行先 DB の DSN を結び付けて migrate を生成します。
	m, err := migrate.New("file://"+migrateFilePlace, driver.DSNWithTimeZoneString(dbCfg, osCfg))
	if err != nil {
		return nil, err
	}
	return m, nil
}

// overrideEnv は、環境変数を一時的に上書きし、元の状態へ戻す復元関数を返します。
func overrideEnv(key, value string) func() {
	prev, existed := os.LookupEnv(key)
	_ = os.Setenv(key, value)
	return func() {
		if existed {
			_ = os.Setenv(key, prev)
			return
		}
		_ = os.Unsetenv(key)
	}
}
