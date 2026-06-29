package migrate

import (
	"errors"
	"fmt"

	"go-boilerplate/internal/logging"
	"go-boilerplate/pkg/xerrors"

	"github.com/golang-migrate/migrate/v4"
)

// MigrateUpRun は、データベースマイグレーションを Up 方向に適用します。
// steps=0 なら全マイグレーションを適用し、正数なら段数分だけ適用します。steps が負の場合はエラーを返します。既に最新の場合（ErrNoChange）は成功扱いです。
func MigrateUpRun(steps int, database string, logger logging.Logger, newMigrator MigratorFactory) error {
	if steps < 0 {
		err := xerrors.New(fmt.Sprintf("steps must be zero or positive, got %d", steps))
		logger.Named("migrateUpRun").Error("invalid steps", logging.Error(logging.ErrorKey, err))
		return err
	}

	m, err := newMigrator(database)
	if err != nil {
		logger.Named("migrateUpRun.buildMigrateInstance").Error("failed to create migrate instance",
			logging.Error(logging.ErrorKey, err),
		)
		return err
	}

	if steps == 0 {
		logger.Named("migrateUpRun").Info("running full migration up")
	} else {
		logger.Named("migrateUpRun").Info("running migration up steps", logging.Int("steps", steps))
	}
	if err := executeMigrateUp(m, steps); err != nil {
		logger.Named("migrateUpRun.executeMigrateUp").Error("migration failed",
			logging.Error(logging.ErrorKey, err),
		)
		return err
	}
	logger.Named("migrateUpRun").Info("✅ migration completed")

	return nil
}

// executeMigrateUp は、steps が 0 なら全件、正なら段数指定で Up します。無変更（ErrNoChange）は成功扱いです。
func executeMigrateUp(m Migrator, steps int) error {
	var err error
	if steps == 0 {
		err = m.Up()
	} else {
		err = m.Steps(steps)
	}
	// 既に最新であれば ErrNoChange になるため、両経路とも成功扱いとして握りつぶします。
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}
