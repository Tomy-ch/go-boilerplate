// Package migrate は、データベースマイグレーションのコアロジック（適用段数の分岐・無変更許容・dirty 復旧）を提供します。
package migrate

//go:generate mockgen -source=$GOFILE -destination=mock/mock_common.gen.go -package=mock_$GOPACKAGE

// Migrator は、golang-migrate のマイグレーション操作（Up / Down / Steps / Version / Force）を抽象化します。
type Migrator interface {
	Up() error
	Down() error
	Steps(n int) error
	Version() (version uint, dirty bool, err error)
	Force(version int) error
}

// MigratorFactory は、対象 DB 名から Migrator を生成する関数型です。
type MigratorFactory func(database string) (Migrator, error)
