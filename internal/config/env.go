package config

// IsLocalClassEnv は、APP_ENV が local 系（local / ci / test）かを返します（deploy 系・未知は false）。
// プールの向き先変更や cmd/db-slot の実行可否など、本番相当で DB を作成/破棄しないためのガードの判定点。
func IsLocalClassEnv(env string) bool {
	switch env {
	case EnvLocal, EnvCI, EnvTest:
		return true
	default:
		return false
	}
}
