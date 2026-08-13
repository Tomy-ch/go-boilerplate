package config

import (
	"bytes"
	"os"
	"reflect" //nolint:depguard // テスト専用。envspec.go のタグから env キー集合を導出するのに reflect が必須で、代替は env ファイルの写経になり検証にならない（本番コードは reflect を使わない）。
	"slices"
	"strings"
	"testing"

	root "go-boilerplate"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//nolint:paralleltest // Load は os.Setenv でグローバル環境を変更するため並列化不可
func TestLoad(t *testing.T) {
	restoreEnvAfterTest(t)

	t.Run("正常系", func(t *testing.T) {
		t.Run("埋め込み env を読み込み環境変数へ反映する", func(t *testing.T) {
			_ = os.Unsetenv("APP_NAME")

			require.NoError(t, Load())
			assert.Equal(t, "Boilerplate", os.Getenv("APP_NAME"))
		})

		t.Run("既存の環境変数は上書きしない", func(t *testing.T) {
			t.Setenv("APP_NAME", "existing-value")

			require.NoError(t, Load())
			assert.Equal(t, "existing-value", os.Getenv("APP_NAME"))
		})

		t.Run("埋め込み env の APP_ENV 素性をパッケージ変数へ捕捉する", func(t *testing.T) {
			// 期待値をハードコードすると別キーへの捕捉ズレを検出できないため、埋め込み env から再導出する。
			t.Setenv("APP_ENV", "injected-at-runtime")
			prev := embeddedAppEnv
			t.Cleanup(func() { embeddedAppEnv = prev })
			embeddedAppEnv = ""

			require.NoError(t, Load())
			assert.Equal(t, parseEmbeddedEnv(t)["APP_ENV"], embeddedAppEnv)
		})
	})
}

func TestEmbeddedEnvConsistency(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("既定値を持たないキーが埋め込み env に全て存在する", func(t *testing.T) {
			t.Parallel()

			embedded := parseEmbeddedEnv(t)

			var required, missing []string
			for key, hasDefault := range declaredEnvKeys(t) {
				if hasDefault {
					continue
				}
				required = append(required, key)
				if _, ok := embedded[key]; !ok {
					missing = append(missing, key)
				}
			}
			require.NotEmpty(t, required, "envDefault 無しのキーが1つも無く、この検証は空振りする")

			slices.Sort(missing)
			assert.Emptyf(t, missing,
				"envspec.go が envDefault 無しで宣言するキーが %s に無い", embeddedEnvFile)
		})

		t.Run("埋め込み env のキーが全て envspec に宣言されている", func(t *testing.T) {
			t.Parallel()

			declared := declaredEnvKeys(t)

			var undeclared []string
			for key := range parseEmbeddedEnv(t) {
				if _, ok := declared[key]; !ok {
					undeclared = append(undeclared, key)
				}
			}

			slices.Sort(undeclared)
			assert.Emptyf(t, undeclared,
				"%s のキーが envspec.go に宣言されていない", embeddedEnvFile)
		})

		t.Run("埋め込み env の OBS_TARGET_STATUS_CODES がテストの期待値と一致する", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, expectedObservabilityTargetStatusCodesStr,
				parseEmbeddedEnv(t)["OBS_TARGET_STATUS_CODES"])
		})
	})
}

// parseEmbeddedEnv は、埋め込み env ファイルをパースしたキーと値を返します。
func parseEmbeddedEnv(t *testing.T) map[string]string {
	t.Helper()

	b, err := root.FS.ReadFile(embeddedEnvFile)
	require.NoError(t, err)
	kv, err := godotenv.Parse(bytes.NewReader(b))
	require.NoError(t, err)
	require.NotEmpty(t, kv)

	return kv
}

// declaredEnvKeys は、Loader が宣言する環境変数キーから envDefault タグの有無への対応を返します。
// envDefault を持つキーは env ファイルへの記載を省いてよく、持たないキーは記載が要ります
// （env/README.md の Conventions）。
func declaredEnvKeys(t *testing.T) map[string]bool {
	t.Helper()

	keys := map[string]bool{}
	collectEnvKeys(t, reflect.TypeFor[Loader](), "", keys)
	require.NotEmpty(t, keys)

	return keys
}

// collectEnvKeys は、typ の env タグを prefix 付きで keys へ集めます。envPrefix を持つ構造体フィールドは
// その prefix を連ねて再帰します。
func collectEnvKeys(t *testing.T, typ reflect.Type, prefix string, keys map[string]bool) {
	t.Helper()

	for field := range typ.Fields() {
		if p, ok := field.Tag.Lookup("envPrefix"); ok {
			require.Equal(t, reflect.Struct, field.Type.Kind(),
				"envPrefix を持つ %s が構造体でない", field.Name)
			collectEnvKeys(t, field.Type, prefix+p, keys)
			continue
		}

		tag, ok := field.Tag.Lookup("env")
		if !ok {
			continue
		}
		name, _, _ := strings.Cut(tag, ",")
		// env:"-" は caarlos0/env が読み飛ばすため、env ファイルの記載対象にもならない。
		if name == "-" {
			continue
		}
		_, hasDefault := field.Tag.Lookup("envDefault")
		keys[prefix+name] = hasDefault
	}
}

// restoreEnvAfterTest は、テスト終了時に環境変数をテスト開始時点へ戻します。
func restoreEnvAfterTest(t *testing.T) {
	t.Helper()
	snapshot := os.Environ()
	t.Cleanup(func() {
		os.Clearenv()
		for _, kv := range snapshot {
			k, v, _ := strings.Cut(kv, "=")
			//nolint:usetesting // テスト後の環境変数一括復元のため t.Setenv は使用できない
			_ = os.Setenv(k, v)
		}
	})
}
