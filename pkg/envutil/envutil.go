// Package envutil は、環境変数を扱うための小さなユーティリティを提供します。
package envutil

import (
	"fmt"
	"os"
)

// Override は、環境変数を一時的に上書きし、元の状態へ戻す復元関数を返します。
//
// 設定読み込みの間だけ特定の環境変数（例: DB_NAME）を差し替える等、プロセス環境のグローバル汚染を
// 防ぎつつ冪等性を保つために使います。上書きに失敗した場合（不正なキー等）はエラーを返し、副作用を
// 残しません。復元関数は、元値が存在した場合はその値へ、存在しなかった場合は Unset へ戻します
// （復元は defer 実行のため best-effort です）。
//
//	restore, err := envutil.Override("DB_NAME", "test")
//	if err != nil {
//		return err
//	}
//	defer restore()
func Override(key, value string) (func(), error) {
	prev, existed := os.LookupEnv(key)
	if err := os.Setenv(key, value); err != nil {
		return func() {}, fmt.Errorf("failed to set env %q: %w", key, err)
	}
	restore := func() {
		if existed {
			_ = os.Setenv(key, prev)
			return
		}
		_ = os.Unsetenv(key)
	}
	return restore, nil
}
