package config

// IsLocalClassEnv は、APP_ENV が local 系（local / ci / test）かを返します（deploy 系・未知は false）。
//
// dast は非 deploy 系ですが false です。ここが判定しているのは「非本番か」ではなく
// 「DB スロットプールに参加させてよいか」で、dast は CI が DB を直接用意するため参加しません。
// 新しい env を足すときは、非本番であることだけを理由に true 側へ入れないこと。
func IsLocalClassEnv(env string) bool {
	switch env {
	case EnvLocal, EnvCI, EnvTest:
		return true
	default:
		return false
	}
}
