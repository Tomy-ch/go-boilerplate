//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE -package=mock_$GOPACKAGE

// Package job は、ジョブを管理・実行するためのコマンドを提供するためのパッケージです。
package job

import (
	"context"
)

// Job は、ジョブを表すインターフェースです。
type Job interface {
	// Name は、ジョブの名前を返します。
	Name() string
	// Execute は、ジョブを実行します。
	Execute(ctx context.Context, args []string) error
}

// Runner は、ジョブの実行を管理するインターフェースです。
type Runner interface {
	// Run は、指定されたジョブを実行します。
	Run(ctx context.Context, jobName string, args []string) error
	// Names は、登録されているジョブの名前一覧を返します。
	Names() []string
}

// State は、ジョブの状態を管理するインターフェースです。
type State interface {
	// Set は、ジョブの状態を設定します。
	Set(name string, args []string, done chan error)
	// Snapshot は、現在のジョブの状態をスナップショットとして取得します。
	Snapshot() (string, []string, chan error)
}
