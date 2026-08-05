package architest

import (
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	pkgexec "go-boilerplate/pkg/exec"
	pkgfs "go-boilerplate/pkg/fs"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// targetStatusCodesKey は、絞り込みの形まで宣言された環境別ポリシーを持つキーです。
const targetStatusCodesKey = "OBS_TARGET_STATUS_CODES"

// envLocalFile は、ローカル実効値のうち env ファイル側の出所です。
const envLocalFile = "env/.env"

// appEnvKey / envLocalProvenance は、env/.env がローカル既定のままかを判別する組です。
// CI は埋め込みのため env/.env を env/.env.<APP_ENV> で上書きするので、作業ツリーの内容が
// ローカル既定であるとは限りません。
const (
	appEnvKey          = "APP_ENV"
	envLocalProvenance = "local"
)

// envValueFiles は、値を突き合わせる env ファイルの全件です（ローカル既定と各環境ファイル）。
var envValueFiles = []string{envLocalFile, "env/.env.ci", "env/.env.dev", "env/.env.stg", "env/.env.prd"}

// targetStatusCodesPolicy は、OBS_TARGET_STATUS_CODES の環境別ポリシーの宣言です。
// 上の段ほど広く監視し、下の段ほど本番の監視ノイズを避けて絞る単調な絞り込みです。
//
// 各段の期待値は 1 つ上の段の実値から excluded を引いて導出するため、環境間で共通のステータスコードは
// この宣言に写経されません。新しいコードを一部の env ファイルにだけ足す伝播漏れは、導出した期待値との
// 差として失敗します。特定の環境から意図的に外す場合に限り excluded の更新が要り、環境別ポリシーの
// 変更が人手の確認を経ることになります。
var targetStatusCodesPolicy = []statusCodeTier{
	{files: []string{"env/.env", "env/.env.ci"}},
	{files: []string{"env/.env.dev", "env/.env.stg"}, excluded: []string{"429"}},
	{files: []string{"env/.env.prd"}, excluded: []string{"403", "404", "405"}},
}

// statusCodeTier は、OBS_TARGET_STATUS_CODES の環境別ポリシーにおける 1 段を表します。
// files は同じ段に属する env ファイル（互いに同値であること）、excluded は 1 つ上の段から
// 落とすステータスコードです。
type statusCodeTier struct {
	files    []string
	excluded []string
}

// TestEnvTargetStatusCodesPolicy は、OBS_TARGET_STATUS_CODES の値が env ファイル間で宣言された
// 環境別ポリシーどおりに分岐していることを機械検証します。
// env ファイルは互いに独立した手書きテキストで、キーの値が環境ごとに違ってよいのか揃うべきなのかを
// 表現する場所が無く、伝播漏れと意図的なポリシーが同じ見た目になります。本テストがその区別を担い、
// 「一部の env にだけ足した」も「揃えるべきでない環境を揃えた」も loud な失敗に変えます。
func TestEnvTargetStatusCodesPolicy(t *testing.T) {
	t.Parallel()

	root := moduleRoot(t)
	requireEnvValueFilesCoverDir(t, root)
	require.ElementsMatch(t, envValueFiles, policyEnvFiles(),
		"環境別ポリシーの段が挙げる env ファイルが envValueFiles と一致せず、対象から漏れた env ファイルがある")

	var upper []string
	for i, tier := range targetStatusCodesPolicy {
		want := upper
		if i > 0 {
			want = excludeStatusCodes(t, upper, tier.excluded)
		}

		for _, file := range tier.files {
			got := readStatusCodes(t, root, file)
			if i == 0 {
				require.NotEmptyf(t, got, "%s の %s が空で、以降の段の検証が空振りする", file, targetStatusCodesKey)
				continue
			}
			assert.Equalf(t, want, got,
				"%s の %s が環境別ポリシー（1 つ上の段から %s を除外）と一致しない。"+
					"意図的な変更なら targetStatusCodesPolicy を更新すること",
				file, targetStatusCodesKey, strings.Join(tier.excluded, ","))
		}
		upper = readStatusCodes(t, root, tier.files[0])
	}
}

// policyEnvFiles は、環境別ポリシーの全段が挙げる env ファイルを段の順に返します。
func policyEnvFiles() []string {
	var files []string
	for _, tier := range targetStatusCodesPolicy {
		files = append(files, tier.files...)
	}
	return files
}

// readEnvFileValues は、env ファイル群をキーごとの「ファイル → 値」へ読み替えて返します。
// env/.env だけは readLocalEnv を通します。CI は埋め込みのため作業ツリーの env/.env を対象環境の
// ファイルで上書きしており、そのまま読むと local と当該環境の差が消えて検証が空振りするためです。
func readEnvFileValues(t *testing.T, root string) map[string]map[string]string {
	t.Helper()

	requireEnvValueFilesCoverDir(t, root)

	values := map[string]map[string]string{}
	for _, file := range envValueFiles {
		kv := parseEnvFile(t, root, file)
		if file == envLocalFile {
			kv = readLocalEnv(t, root)
		}

		for key, value := range kv {
			if values[key] == nil {
				values[key] = map[string]string{}
			}
			values[key][file] = value
		}
	}

	return values
}

// requireEnvValueFilesCoverDir は、envValueFiles が env/ 配下の env ファイル実体を網羅していることを
// 確かめます。env ファイルは増える方向にしか壊れません（減れば読み込みが失敗する）。新しい環境の
// ファイルを足して一覧への追記を忘れると、そのファイルだけ検証対象から静かに外れます。
func requireEnvValueFilesCoverDir(t *testing.T, root string) {
	t.Helper()

	found, err := pkgfs.OS{}.Glob(filepath.Join(root, "env", ".env*"))
	require.NoError(t, err, "env ファイルの一覧を取得できない")

	onDisk := make([]string, 0, len(found))
	for _, path := range found {
		onDisk = append(onDisk, filepath.ToSlash(filepath.Join("env", filepath.Base(path))))
	}

	assert.ElementsMatch(t, envValueFiles, onDisk,
		"env/ 配下の env ファイルと envValueFiles が一致しない。環境を追加したなら envValueFiles にも追記すること")
}

// readStatusCodes は、env ファイルの OBS_TARGET_STATUS_CODES をカンマ区切りで分解して返します。
func readStatusCodes(t *testing.T, root, file string) []string {
	t.Helper()

	value, ok := parseEnvFile(t, root, file)[targetStatusCodesKey]
	require.Truef(t, ok, "%s に %s が無い", file, targetStatusCodesKey)
	return strings.Split(value, ",")
}

// excludeStatusCodes は、codes から excluded を除いた並びを返します。excluded に codes へ含まれない
// コードがあれば、ポリシー宣言が陳腐化して除外が空振りしているため失敗させます。
func excludeStatusCodes(t *testing.T, codes, excluded []string) []string {
	t.Helper()

	for _, code := range excluded {
		require.Containsf(t, codes, code,
			"ポリシーが除外する %s が 1 つ上の段に無く、除外指定が空振りしている", code)
	}

	out := make([]string, 0, len(codes))
	for _, code := range codes {
		if !slices.Contains(excluded, code) {
			out = append(out, code)
		}
	}
	return out
}

// readLocalEnv は、ローカル既定の env 値を返します。CI は埋め込みのため env/.env を
// env/.env.<APP_ENV> で上書きする（make materialize-env）ので、作業ツリーの APP_ENV が
// ローカルでなければコミット済みの内容を読み直します。
func readLocalEnv(t *testing.T, root string) map[string]string {
	t.Helper()

	values := parseEnvContent(t, envLocalFile, readRepoFile(t, root, envLocalFile))
	if values[appEnvKey] == envLocalProvenance {
		return values
	}
	return parseEnvContent(t, envLocalFile, readCommittedFile(t, root, envLocalFile))
}

// readCommittedFile は、リポジトリにコミットされた時点のファイル内容を返します。
func readCommittedFile(t *testing.T, root, file string) string {
	t.Helper()

	out, err := pkgexec.OS{}.Output(t.Context(), root, nil, "git", []string{"show", "HEAD:" + file})
	require.NoErrorf(t, err, "%s のコミット済み内容を git から取得できない", file)
	return string(out)
}

// parseEnvFile は、env ファイルをキーと値の対応へ分解して返します。
func parseEnvFile(t *testing.T, root, file string) map[string]string {
	t.Helper()

	return parseEnvContent(t, file, readRepoFile(t, root, file))
}

// parseEnvContent は、env ファイルの内容をキーと値の対応へ分解して返します。
// アプリ本体のローダー（internal/config）と同じ godotenv で解釈します。独自パーサだと
// クォートやコメントの扱いが分かれ、テストだけが実行時と違う値を見ることになります。
func parseEnvContent(t *testing.T, file, content string) map[string]string {
	t.Helper()

	values, err := godotenv.Parse(strings.NewReader(content))
	require.NoErrorf(t, err, "%s を dotenv として解釈できない", file)
	require.NotEmptyf(t, values, "%s から代入行を 1 件も抽出できず、検証が空振りする", file)
	return values
}

func Test_parseEnvContent(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("代入行をキーと値の対応へ分解する", func(t *testing.T) {
			t.Parallel()

			values := parseEnvContent(t, "env/.env", "APP_ENV=local\nOBS_TARGET_STATUS_CODES=403,404\n")

			assert.Equal(t,
				map[string]string{"APP_ENV": "local", "OBS_TARGET_STATUS_CODES": "403,404"}, values)
		})

		t.Run("コメント行と空行を値として扱わない", func(t *testing.T) {
			t.Parallel()

			values := parseEnvContent(t, "env/.env", "# 環境\n\nAPP_ENV=local\n")

			assert.Equal(t, map[string]string{"APP_ENV": "local"}, values)
		})

		t.Run("クォートで囲んだ値からクォートを外す", func(t *testing.T) {
			t.Parallel()

			values := parseEnvContent(t, "env/.env", "OBS_TARGET_STATUS_CODES=\"403,404\"\n")

			assert.Equal(t, map[string]string{"OBS_TARGET_STATUS_CODES": "403,404"}, values)
		})

		t.Run("export 前置の代入行も分解する", func(t *testing.T) {
			t.Parallel()

			values := parseEnvContent(t, "env/.env", "export APP_ENV=local\n")

			assert.Equal(t, map[string]string{"APP_ENV": "local"}, values)
		})
	})
}

func Test_excludeStatusCodes(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("指定したコードを除いた並びを元の順序のまま返す", func(t *testing.T) {
			t.Parallel()

			got := excludeStatusCodes(t, []string{"403", "404", "405", "429"}, []string{"404", "429"})

			assert.Equal(t, []string{"403", "405"}, got)
		})

		t.Run("除外が空なら元の並びをそのまま返す", func(t *testing.T) {
			t.Parallel()

			got := excludeStatusCodes(t, []string{"403", "404"}, nil)

			assert.Equal(t, []string{"403", "404"}, got)
		})

		t.Run("全件を除外すれば空になる", func(t *testing.T) {
			t.Parallel()

			assert.Empty(t, excludeStatusCodes(t, []string{"403"}, []string{"403"}))
		})
	})
}

func Test_policyEnvFiles(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("全段のenvファイルを段の順に平坦化して返す", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t,
				[]string{"env/.env", "env/.env.ci", "env/.env.dev", "env/.env.stg", "env/.env.prd"},
				policyEnvFiles())
		})
	})
}

// readRepoFile は、モジュールルートからの相対パスでリポジトリ内のファイルを読みます。
func readRepoFile(t *testing.T, root, file string) string {
	t.Helper()

	b, err := pkgfs.OS{}.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
	require.NoErrorf(t, err, "%s を読めない", file)
	return string(b)
}

// moduleRoot は、本テストファイル位置（internal/architest）から辿ったモジュールルートを返します。
func moduleRoot(t *testing.T) string {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Join(filepath.Dir(thisFile), "..", "..")
}
