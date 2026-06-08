// Package envutil は、環境変数を扱うための小さなユーティリティを提供します。
package envutil

import "os"

// Override は、環境変数を一時的に上書きし、元の状態へ戻す復元関数を返します。
//
// 設定読み込みの間だけ特定の環境変数（例: DB_NAME）を差し替える等、プロセス環境のグローバル汚染を
// 防ぎつつ冪等性を保つために使います。復元関数は、元値が存在した場合はその値へ、存在しなかった場合は
// Unset へ戻します。
//
//	restore := envutil.Override("DB_NAME", "test")
//	defer restore()
func Override(key, value string) func() {
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
