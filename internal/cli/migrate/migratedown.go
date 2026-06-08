package migrate

import (
	"errors"
	"fmt"

	"go-boilerplate/internal/logging"
	"go-boilerplate/pkg/safecast"

	"github.com/golang-migrate/migrate/v4"
)

// MigrateDownRun は、マイグレーションをダウングレードするための実行関数です。
func MigrateDownRun(steps int, database string, newMigrator MigratorFactory) error {
	logger, err := logging.NewProductionLogger()
	if err != nil {
		panic("failed to create logger: " + err.Error())
	}

	if steps < 0 {
		// 負値を許すと符号反転で Up 方向へ進んでしまうため、Down コマンドでは弾きます。
		err := fmt.Errorf("steps must be zero or positive, got %d", steps)
		logger.Named("migrateDownRun").Error("invalid steps", logging.Error("validateSteps", err))
		return err
	}

	// CLI オプションを反映した migrate インスタンスを組み立てます。
	m, err := newMigrator(database)
	if err != nil {
		logger.Named("migrateDownRun.buildMigrateInstance").Error("failed to create migrate instance",
			logging.Error("buildMigrateInstance", err),
		)
		return err
	}

	if steps == 0 {
		// 段数未指定なら、現在適用済みの migration を最後まで巻き戻します。
		logger.Named("migrateDownRun").Info("running full migration down")
		if err := executeMigrateFullDown(m); err != nil {
			logger.Named("migrateDownRun.executeMigrateFullDown").Error("down migration failed",
				logging.Error("executeMigrateFullDown", err),
			)
			return err
		}
	} else {
		// 段数指定時は、現在位置から指定段数分だけ Down を進めます。
		logger.Named("migrateDownRun").Info("running migration down steps", logging.Int("steps", steps))
		if err := executeMigrateDownSteps(m, steps); err != nil {
			logger.Named("migrateDownRun.executeMigrateDownSteps").Error("down migration steps failed",
				logging.Error("executeMigrateDownSteps", err),
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
