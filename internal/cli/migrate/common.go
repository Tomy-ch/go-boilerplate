// Package migrate は、データベースマイグレーションのコアロジック（適用段数の分岐・無変更許容・dirty 復旧）を提供します。
package migrate

import "go-boilerplate/pkg/xerrors"

//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// errNegativeSteps は、適用段数に負値が指定された場合のエラーです。
var errNegativeSteps = xerrors.New("steps must be zero or positive")

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
