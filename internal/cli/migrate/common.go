// Package migrate は、データベースマイグレーションのコアロジック（適用段数の分岐・無変更許容・dirty 復旧）を提供します。
package migrate

import (
	"os"
)

//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE -package=mock_$GOPACKAGE

// Migrator は、golang-migrate の操作を抽象化し、テストでモック差し替え可能にします。
type Migrator interface {
	Up() error
	Down() error
	Steps(n int) error
	Version() (version uint, dirty bool, err error)
	Force(version int) error
}

// MigratorFactory は、対象 DB 名から Migrator を生成する関数型です。
type MigratorFactory func(database string) (Migrator, error)

// OverrideEnv は、環境変数を一時的に上書きし、元の状態へ戻す復元関数を返します。
//
// config 読み取り中だけ DB_NAME を差し替える等、プロセス環境のグローバル汚染を防ぎつつ冪等性を保つために使います。
func OverrideEnv(key, value string) func() {
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
