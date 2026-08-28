package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	root "go-boilerplate"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"
)

// authIssuerEnvKey は、JWT の issuer を持つ環境変数のキーです。
const authIssuerEnvKey = "AUTH_ISSUER"

// NewTestLocation は、テスト用のタイムゾーンロケーションを生成します。
func NewTestLocation(t *testing.T) *time.Location {
	t.Helper()
	cfg := MockConfigForTest(t)
	osCfg := NewOperatingSystemConfig(cfg)

	loc, err := NewTimeLocation(osCfg)
	require.NoError(t, err)
	return loc
}

// ResolvedAuthIssuer は、この実行環境の AUTH_ISSUER を返します。解決順は Load と同じく実行時 env が先で、
// 無ければ埋め込み env/.env の値です。issuer は worktree の DB スロットでポートがずれる（実行時 env は
// make が渡す）ため、環境固有の issuer に依存するテストは値をリテラルで固定せずここから取ります。
func ResolvedAuthIssuer(t *testing.T) string {
	t.Helper()

	if v := os.Getenv(authIssuerEnvKey); v != "" {
		return v
	}

	b, err := root.FS.ReadFile(embeddedEnvFile)
	require.NoError(t, err)
	kv, err := godotenv.Parse(bytes.NewReader(b))
	require.NoError(t, err)
	require.NotEmpty(t, kv[authIssuerEnvKey])

	return kv[authIssuerEnvKey]
}

// EnsureRepoRootAndEnv は、go.mod のあるリポジトリルートへ作業ディレクトリを移動し、
// env/.env.<env> の各値を t.Setenv で設定します。埋め込み（local）より対象環境の値が優先されます。
// 作業ディレクトリと環境変数の復元は t.Chdir / t.Setenv が自動で行います。
func EnsureRepoRootAndEnv(t *testing.T, env string) {
	t.Helper()

	root := repoRoot(t)
	t.Chdir(root)

	kv, err := godotenv.Read(filepath.Join(root, "env", ".env."+env))
	require.NoError(t, err)
	for k, v := range kv {
		t.Setenv(k, v)
	}

	setWorktreeDBName(t, env)
}

// setWorktreeDBName は、DB スロットプール利用時に対象 env の DB_NAME を自 worktree のデータベースへ
// 上書きする。local は DB_NAME_LOCAL（wt<N>_local）、ci/test は DB_NAME_TEST（wt<N>_test）。deploy 系
// env では本番 DB を誤指しないよう無視する（IsLocalClassEnv）。未使用時は .env.<env> の値のまま。
func setWorktreeDBName(t *testing.T, env string) {
	t.Helper()
	switch {
	case env == EnvLocal && IsLocalClassEnv(env):
		if v := os.Getenv("DB_NAME_LOCAL"); v != "" {
			t.Setenv("DB_NAME", v)
		}
	case IsLocalClassEnv(env):
		t.Setenv("DB_NAME", testDBName())
	case os.Getenv("DB_NAME_TEST") != "" || os.Getenv("DB_NAME_LOCAL") != "":
		fmt.Fprintf(os.Stderr,
			"[config] 警告: env=%q は local/test 系でないため、DB スロットプールの向き先変更を無視します\n", env)
	}
}

// repoRoot は、go.mod を上方向に探索してリポジトリルートを返します。
func repoRoot(t *testing.T) string {
	t.Helper()

	p, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, err := os.Stat(filepath.Join(p, "go.mod")); err == nil {
			return p
		}
		parent := filepath.Dir(p)
		require.NotEqual(t, parent, p, "リポジトリルート（go.mod）が見つかりません")
		p = parent
	}
}

// Realtime Delivery の contract test が繋ぐ先。既定は共有インフラ / CI service の DynamoDB Local で、
// 環境変数で本番 DynamoDB へ向け直せる（同じテストを両方に対して走らせるため）。
const (
	realtimeTestEndpointEnvKey    = "REALTIME_TEST_ENDPOINT"
	realtimeTestRegionEnvKey      = "REALTIME_TEST_REGION"
	realtimeTestAccessKeyEnvKey   = "REALTIME_TEST_ACCESS_KEY_ID"
	realtimeTestSecretKeyEnvKey   = "REALTIME_TEST_SECRET_ACCESS_KEY" //nolint:gosec // 環境変数名であり資格情報ではない
	defaultRealtimeTestEndpoint   = "http://localhost:8000"
	defaultRealtimeTestRegion     = "us-east-1"
	defaultRealtimeTestCredential = "test"
)

// RealtimeTestConnection は、Realtime Delivery の contract test が使う接続先です。
type RealtimeTestConnection struct {
	// Endpoint は、DynamoDB 互換エンドポイントです。空なら SDK 既定の解決（本番 DynamoDB）です。
	Endpoint string
	// Region は、署名に用いるリージョンです。
	Region string
	// AccessKeyID / SecretAccessKey は、静的資格情報です。両方空なら SDK 既定の chain に委ねます。
	AccessKeyID     string
	SecretAccessKey string
}

// NewRealtimeTestConnection は、contract test の接続先を返します。既定は `http://localhost:8000` の
// DynamoDB Local（静的資格情報はダミー）。REALTIME_TEST_ENDPOINT を空文字で設定し、REALTIME_TEST_REGION を
// 与え、資格情報の 2 変数を空文字で設定すると、同じテストが SDK 既定の chain で本番 DynamoDB に対して走ります
// （「未設定」と「空文字」を区別するため LookupEnv で読む）。
func NewRealtimeTestConnection(t *testing.T) RealtimeTestConnection {
	t.Helper()

	lookup := func(key, fallback string) string {
		if v, ok := os.LookupEnv(key); ok {
			return v
		}

		return fallback
	}

	return RealtimeTestConnection{
		Endpoint:        lookup(realtimeTestEndpointEnvKey, defaultRealtimeTestEndpoint),
		Region:          lookup(realtimeTestRegionEnvKey, defaultRealtimeTestRegion),
		AccessKeyID:     lookup(realtimeTestAccessKeyEnvKey, defaultRealtimeTestCredential),
		SecretAccessKey: lookup(realtimeTestSecretKeyEnvKey, defaultRealtimeTestCredential),
	}
}
