package migrate

import (
	"errors"
	"fmt"

	"go-boilerplate/internal/logging"
	"go-boilerplate/pkg/safecast"
	"go-boilerplate/pkg/xerrors"

	"github.com/golang-migrate/migrate/v4"
)

// MigrateDownRun は、データベースマイグレーションを Down 方向に適用します。
// steps=0 なら全マイグレーションをロールバックし、正数なら段数分だけ戻します。steps が負の場合はエラーを返します。dirty 状態の DB は Force で整合を取ってから Down します。
func MigrateDownRun(steps int, database string, logger logging.Logger, newMigrator MigratorFactory) error {
	if steps < 0 {
		// 負値を許すと符号反転で Up 方向へ進んでしまうため、Down コマンドでは弾きます。
		err := xerrors.New(fmt.Sprintf("steps must be zero or positive, got %d", steps))
		logger.Named("migrateDownRun").Error("invalid steps", logging.Error(logging.ErrorKey, err))
		return err
	}

	m, err := newMigrator(database)
	if err != nil {
		logger.Named("migrateDownRun.buildMigrateInstance").Error("failed to create migrate instance",
			logging.Error(logging.ErrorKey, err),
		)
		return err
	}

	if steps == 0 {
		logger.Named("migrateDownRun").Info("running full migration down")
		if err := executeMigrateFullDown(m); err != nil {
			logger.Named("migrateDownRun.executeMigrateFullDown").Error("down migration failed",
				logging.Error(logging.ErrorKey, err),
			)
			return err
		}
	} else {
		logger.Named("migrateDownRun").Info("running migration down steps", logging.Int("steps", steps))
		if err := executeMigrateDownSteps(m, steps); err != nil {
			logger.Named("migrateDownRun.executeMigrateDownSteps").Error("down migration steps failed",
				logging.Error(logging.ErrorKey, err),
			)
			return err
		}
	}

	logger.Named("migrateDownRun").Info("✅ migration down completed")
	return nil
}

// executeMigrateDownSteps は、現在位置から steps 段だけ Down します。無変更（ErrNoChange）は成功扱いです。
func executeMigrateDownSteps(m Migrator, steps int) error {
	// golang-migrate の Steps は負数を渡すとその段数だけ Down するため、検証済みの正値を反転します。
	if err := m.Steps(-steps); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

// executeMigrateFullDown は、マイグレーションを全てダウングレードして、DBを初期状態に戻します。
func executeMigrateFullDown(m Migrator) error {
	// dirty 状態のままでは Down できないため、現在バージョンで整合を取り直してから巻き戻します。
	v, dirty, err := m.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return err
	}
	if dirty {
		safeValue, err := safecast.UintToInt(v)
		if err != nil {
			return err
		}
		if err := m.Force(safeValue); err != nil {
			return err
		}
	}
	if err := m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}
