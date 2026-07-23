package config

// IsLocalClassEnv は、APP_ENV が local 系（local / ci / test）かを返します。deploy 系（dev/stg/prd）
// および未知ラベルは false（deny 型）。DB スロットプールの動的向き先変更（DB_NAME_TEST）や
// pool CLI（cmd/db-pool）の実行可否ガードに用い、本番相当環境で DB を作成/破棄しないための単一の判定点です。
func IsLocalClassEnv(env string) bool {
	switch env {
	case EnvLocal, EnvCI, EnvTest:
		return true
	default:
		return false
	}
}
