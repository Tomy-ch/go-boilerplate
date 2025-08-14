// Package transaction は、トランザクションの管理を提供します。
package transaction

import (
	"context"
)

// Manager は、トランザクションの管理を行うインターフェースです。
type Manager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

// DoWithResult は、トランザクション内で値を取得するためのヘルパー関数です。
func DoWithResult[T any](
	ctx context.Context, m Manager, fn func(ctx context.Context) (T, error),
) (T, error) {
	var result T
	if err := m.Do(ctx, func(ctx context.Context) error {
		var err error
		result, err = fn(ctx)
		return err
	}); err != nil {
		var zero T
		return zero, err
	}
	return result, nil
}
