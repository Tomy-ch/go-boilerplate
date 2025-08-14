// Package migrate は、データベースのマイグレーションに関する機能を提供します。
package migrate

import (
	"boilerplate-go/internal/config"

	"github.com/golang-migrate/migrate/v4"
)

const (
	// migrateFilePlace は、マイグレーションファイルの場所を定義します。
	migrateFilePlace = "database/migrations"
)

// buildMigrateInstance は、マイグレーションインスタンスを生成します。
func buildMigrateInstance() (*migrate.Migrate, error) {
	cfg, err := config.SetUpConfig()
	if err != nil {
		return nil, err
	}
	return migrate.New("file://"+migrateFilePlace, cfg.DatabaseDSN())
}
