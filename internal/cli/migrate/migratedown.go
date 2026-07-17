package migrate

import (
	"context"
	"fmt"

	"go-boilerplate/internal/logging"
	"go-boilerplate/pkg/safecast"
	"go-boilerplate/pkg/xerrors"

	"github.com/golang-migrate/migrate/v4"
)

// MigrateDownRun は、データベースマイグレーションを Down 方向に適用します。
// steps=0 なら全マイグレーションをロールバックし、正数なら段数分だけ戻します。steps が負の場合はエラーを返します。dirty 状態の DB は Force で整合を取ってから Down します。
func MigrateDownRun(steps int, database string, logger logging.Logger, newMigrator MigratorFactory) error {
	ctx := context.Background()

	if steps < 0 {
		// 負値を許すと符号反転で Up 方向へ進んでしまうため、Down コマンドでは弾きます。
		err := xerrors.New(fmt.Sprintf("steps must be zero or positive, got %d", steps))
		logger.Named("migrateDownRun").Error(ctx, "invalid steps", logging.Error(logging.ErrorKey, err))
		return err
	}

	m, err := newMigrator(database)
	if err != nil {
		logger.Named("migrateDownRun.buildMigrateInstance").Error(ctx, "failed to create migrate instance",
			logging.Error(logging.ErrorKey, err),
		)
		return err
	}

	if steps == 0 {
		logger.Named("migrateDownRun").Info(ctx, "running full migration down")
		if err := executeMigrateFullDown(m); err != nil {
			logger.Named("migrateDownRun.executeMigrateFullDown").Error(ctx, "down migration failed",
				logging.Error(logging.ErrorKey, err),
			)
			return err
		}
	} else {
		logger.Named("migrateDownRun").Info(ctx, "running migration down steps", logging.Int("steps", steps))
		if err := executeMigrateDownSteps(m, steps); err != nil {
			logger.Named("migrateDownRun.executeMigrateDownSteps").Error(ctx, "down migration steps failed",
				logging.Error(logging.ErrorKey, err),
			)
			return err
		}
	}

	logger.Named("migrateDownRun").Info(ctx, "✅ migration down completed")
	return nil
}

// executeMigrateDownSteps は、現在位置から steps 段だけ Down します。無変更（ErrNoChange）は成功扱いです。
func executeMigrateDownSteps(m Migrator, steps int) error {
	// golang-migrate の Steps は負数を渡すとその段数だけ Down するため、検証済みの正値を反転します。
	if err := m.Steps(-steps); err != nil && !xerrors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

// executeMigrateFullDown は、マイグレーションを全てダウングレードして、DBを初期状態に戻します。
func executeMigrateFullDown(m Migrator) error {
	// dirty 状態のままでは Down できないため、現在バージョンで整合を取り直してから巻き戻します。
	v, dirty, err := m.Version()
	if err != nil && !xerrors.Is(err, migrate.ErrNilVersion) {
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
	if err := m.Down(); err != nil && !xerrors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}
