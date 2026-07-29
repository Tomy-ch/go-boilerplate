package architest

import (
	"fmt"
	"maps"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"testing"

	pkgfs "go-boilerplate/pkg/fs"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// envReadmeFile は、env 変数一覧の正本（英語）です。日本語版 README.ja.md は正本の翻訳であり、
// 対訳同期は別の手当てに委ねるため検証対象に含めません。
const envReadmeFile = "env/README.md"

// envSpecFile は、env 契約の SSOT である Loader 構造体の宣言元です。
const envSpecFile = "internal/config/envspec.go"

// targetStatusCodesKey は、絞り込みの形まで宣言された環境別ポリシーを持つキーです。
const targetStatusCodesKey = "OBS_TARGET_STATUS_CODES"

// perEnvValueMarker は、env ファイル間で値が異なることを宣言する Notes 列のマーカーです。
const perEnvValueMarker = "Per-environment value"

var (
	// envValueFiles は、値を突き合わせる env ファイルの全件です（ローカル既定と各環境ファイル）。
	envValueFiles = []string{"env/.env", "env/.env.ci", "env/.env.dev", "env/.env.stg", "env/.env.prd"}
	// loaderPrefixRe は、Loader のサブシステムフィールドから型名と envPrefix を捕捉します。
	loaderPrefixRe = regexp.MustCompile("^\\s+\\w+\\s+(\\w+)\\s+`envPrefix:\"([^\"]*)\"`$")
	// structDeclRe は、構造体宣言 type <名前> struct { から名前を捕捉します。
	structDeclRe = regexp.MustCompile(`^type (\w+) struct \{$`)
	// envTagRe / envDefaultRe は、フィールドタグから env キー名と既定値を捕捉します。
	envTagRe     = regexp.MustCompile(`env:"([^",]+)`)
	envDefaultRe = regexp.MustCompile(`envDefault:"([^"]*)"`)
	// readmeRowRe は、env 変数表の行からキーを捕捉します。
	readmeRowRe = regexp.MustCompile(`^\|([A-Z][A-Z0-9_]+)\|`)
	// codeDefaultRe は、Notes 列の Code default 記法からバッククォートで囲まれた値を捕捉します。
	codeDefaultRe = regexp.MustCompile("Code default `([^`]*)`")
)

// targetStatusCodesPolicy は、OBS_TARGET_STATUS_CODES の環境別ポリシーの宣言です。
// 上の段ほど広く監視し、下の段ほど本番の監視ノイズを避けて絞る単調な絞り込みで、その根拠は
// env/README.md の同キーの Notes 列にあります。
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

// readmeRow は、env 変数表の 1 行から取り出した Example 列と Notes 列です。
type readmeRow struct {
	example string
	notes   string
}

// TestEnvTargetStatusCodesPolicy は、OBS_TARGET_STATUS_CODES の値が env ファイル間で宣言された
// 環境別ポリシーどおりに分岐していることを機械検証します。
// env ファイルは互いに独立した手書きテキストで、キーの値が環境ごとに違ってよいのか揃うべきなのかを
// 表現する場所が無く、伝播漏れと意図的なポリシーが同じ見た目になります。本テストがその区別を担い、
// 「一部の env にだけ足した」も「揃えるべきでない環境を揃えた」も loud な失敗に変えます。
func TestEnvTargetStatusCodesPolicy(t *testing.T) {
	t.Parallel()

	values := readEnvFileValues(t, moduleRoot(t))

	var upper []string
	for i, tier := range targetStatusCodesPolicy {
		want := upper
		if i > 0 {
			want = excludeStatusCodes(t, upper, tier.excluded)
		}

		for _, file := range tier.files {
			got := readStatusCodes(t, values, file)
			if i == 0 {
				require.NotEmptyf(t, got, "%s の %s が空で、以降の段の検証が空振りする", file, targetStatusCodesKey)
				continue
			}
			assert.Equalf(t, want, got,
				"%s の %s が環境別ポリシー（1 つ上の段から %s を除外）と一致しない。"+
					"意図的な変更なら targetStatusCodesPolicy と env/README.md の Notes を更新すること",
				file, targetStatusCodesKey, strings.Join(tier.excluded, ","))
		}
		upper = readStatusCodes(t, values, tier.files[0])
	}
}

// TestEnvPerEnvironmentValuePolicy は、env ファイル間で値が割れるキーの集合と、env/README.md の
// Notes 列に置かれた環境差マーカーの集合が双方向に一致することを機械検証します。
// env ファイルは互いに独立した手書きテキストで、値が環境ごとに違ってよいのか揃うべきなのかを
// 表現する場所が無く、伝播漏れと意図的なポリシーが同じ見た目になります。README の Notes に
// 書かれた宣言だけがその区別を持つため、マーカーの無いキーが割れている状態（伝播漏れ）と、
// 値が揃ったのにマーカーが残った状態（陳腐化した宣言）の双方を loud な失敗に変えます。
func TestEnvPerEnvironmentValuePolicy(t *testing.T) {
	t.Parallel()

	root := moduleRoot(t)
	rows := parseEnvReadmeRows(t, root)
	values := readEnvFileValues(t, root)

	for _, key := range slices.Sorted(maps.Keys(values)) {
		row, ok := rows[key]
		if !assert.Truef(t, ok, "%s が env ファイルにあるが %s の変数表に無い", key, envReadmeFile) {
			continue
		}

		split := describeValueSplit(values[key])
		if strings.Contains(row.notes, perEnvValueMarker) {
			assert.NotEmptyf(t, split,
				"%s の Notes は %q と宣言しているが、値は全 env ファイルで一致している。"+
					"宣言が陳腐化しているのでマーカーを外すこと", key, perEnvValueMarker)
			continue
		}
		assert.Emptyf(t, split,
			"%s の値が env ファイル間で割れている（%s）が、%s の Notes に理由が書かれていない。"+
				"伝播漏れなら値を揃え、意図的なら %q と理由を Notes に書くこと",
			key, strings.Join(split, " / "), envReadmeFile, perEnvValueMarker)
	}

	// env ファイルから消えたキーは上のループに入らないため、マーカーだけが残った行を別に拾います。
	for _, key := range slices.Sorted(maps.Keys(rows)) {
		if !strings.Contains(rows[key].notes, perEnvValueMarker) {
			continue
		}
		_, declared := values[key]
		assert.Truef(t, declared,
			"%s の Notes は %q と宣言しているが、キーがどの env ファイルにも無い。"+
				"宣言が陳腐化しているのでマーカーを外すこと", key, perEnvValueMarker)
	}
}

// TestEnvReadmeTargetStatusCodesExample は、env/README.md の Example 列が env/.env の実値と
// 一致することを検証します。Example 列はローカル既定を載せる規約ですが、実体は env ファイルの
// 複製であり、片方だけ更新しても何も検知しません。
func TestEnvReadmeTargetStatusCodesExample(t *testing.T) {
	t.Parallel()

	root := moduleRoot(t)
	values := readEnvFileValues(t, root)

	row, ok := parseEnvReadmeRows(t, root)[targetStatusCodesKey]
	require.Truef(t, ok, "%s の行が %s に無い", targetStatusCodesKey, envReadmeFile)
	assert.Equalf(t, strings.Join(readStatusCodes(t, values, "env/.env"), ","), row.example,
		"%s の Example 列が env/.env の実値と一致しない", targetStatusCodesKey)
}

// TestEnvReadmeCodeDefaults は、env/README.md の Notes 列にある Code default の記載が
// internal/config/envspec.go の envDefault タグと双方向に一致することを検証します。
// Notes 列の記載はタグの複製であり、既定値を変えて README を直し忘れれば、運用者は存在しない
// 既定値を読むことになります。
func TestEnvReadmeCodeDefaults(t *testing.T) {
	t.Parallel()

	root := moduleRoot(t)
	declared := parseEnvDefaults(t, root)
	documented := collectCodeDefaults(t, root)

	for key, want := range declared {
		got, ok := documented[key]
		if !assert.Truef(t, ok, "%s は envDefault:%q を持つが %s に Code default の記載が無い", key, want, envReadmeFile) {
			continue
		}
		assert.Equalf(t, want, got, "%s の Code default が envspec.go の envDefault と一致しない", key)
	}

	for key := range documented {
		_, ok := declared[key]
		assert.Truef(t, ok, "%s は %s に Code default と記載されているが envspec.go に envDefault が無い", key, envReadmeFile)
	}
}

// readStatusCodes は、env ファイルの OBS_TARGET_STATUS_CODES をカンマ区切りで分解して返します。
func readStatusCodes(t *testing.T, values map[string]map[string]string, file string) []string {
	t.Helper()

	value, ok := values[targetStatusCodesKey][file]
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

// readEnvFileValues は、env ファイル群をキーごとの「ファイル → 値」へ読み替えて返します。
// 解釈は godotenv に委ね、末尾コメントの扱いを含めて実行時ローダー（internal/config）と揃えます。
func readEnvFileValues(t *testing.T, root string) map[string]map[string]string {
	t.Helper()

	requireEnvValueFilesCoverDir(t, root)

	values := map[string]map[string]string{}
	for _, file := range envValueFiles {
		kv, err := godotenv.Parse(strings.NewReader(readRepoFile(t, root, file)))
		require.NoErrorf(t, err, "%s を env として解釈できない", file)
		require.NotEmptyf(t, kv, "%s からキーを 1 件も読めず、検証が空振りする", file)

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

// describeValueSplit は、1 つのキーの値が env ファイル間で割れているときにその内訳を返します。
// 値を宣言しないファイルは比較に含めません（deploy 環境が secret manager から受け取るキーは
// env ファイルに現れず、不在は値の差ではないため）。揃っていれば nil を返します。
func describeValueSplit(byFile map[string]string) []string {
	distinct := map[string]struct{}{}
	for _, value := range byFile {
		distinct[value] = struct{}{}
	}
	if len(distinct) <= 1 {
		return nil
	}

	split := make([]string, 0, len(byFile))
	for _, file := range envValueFiles {
		if value, ok := byFile[file]; ok {
			split = append(split, fmt.Sprintf("%s=%q", file, value))
		}
	}
	return split
}

// parseEnvReadmeRows は、env/README.md の変数表をキーごとの行へ分解して返します。
func parseEnvReadmeRows(t *testing.T, root string) map[string]readmeRow {
	t.Helper()

	rows := map[string]readmeRow{}
	for line := range strings.Lines(readRepoFile(t, root, envReadmeFile)) {
		line = strings.TrimRight(line, "\n")
		m := readmeRowRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		// 行頭と行末の区切りにより前後へ空要素が付くため、Example は 5 番目、Notes は末尾の空要素を除く残り。
		cells := strings.Split(line, "|")
		require.GreaterOrEqualf(t, len(cells), 7, "%s の行の列数が想定と異なる: %s", envReadmeFile, line)
		rows[m[1]] = readmeRow{
			example: cells[4],
			notes:   strings.Join(cells[5:len(cells)-1], "|"),
		}
	}

	require.NotEmptyf(t, rows, "%s から変数表の行を 1 件も抽出できず、検証が空振りする", envReadmeFile)
	return rows
}

// collectCodeDefaults は、env/README.md の Notes 列に Code default を持つキーとその値を返します。
// 値が空であることは Code default empty と綴る規約で、バッククォート記法とは別に受け付けます。
func collectCodeDefaults(t *testing.T, root string) map[string]string {
	t.Helper()

	defaults := map[string]string{}
	for key, row := range parseEnvReadmeRows(t, root) {
		if !strings.Contains(row.notes, "Code default") {
			continue
		}
		if m := codeDefaultRe.FindStringSubmatch(row.notes); m != nil {
			defaults[key] = m[1]
			continue
		}
		require.Containsf(t, row.notes, "Code default empty",
			"%s の Code default が既定の記法（バッククォート囲み もしくは empty）で書かれていない", key)
		defaults[key] = ""
	}

	require.NotEmptyf(t, defaults, "%s から Code default を 1 件も抽出できず、検証が空振りする", envReadmeFile)
	return defaults
}

// parseEnvDefaults は、envspec.go が envDefault タグ付きで宣言する env キーと既定値を返します。
// キーは Loader のサブシステムフィールドが持つ envPrefix を連ねた完全形にします。
// depguard が go/ast を禁じるため、gofmt 済みソースのテキスト走査で抽出します（既存 architest と同方針）。
func parseEnvDefaults(t *testing.T, root string) map[string]string {
	t.Helper()

	lines := strings.Split(readRepoFile(t, root, envSpecFile), "\n")
	prefixes := collectEnvPrefixes(lines)
	require.NotEmpty(t, prefixes, "Loader から envPrefix を 1 件も抽出できず、検証が空振りする")

	defaults := map[string]string{}
	current := ""
	for _, line := range lines {
		if m := structDeclRe.FindStringSubmatch(line); m != nil {
			current = prefixes[m[1]]
			continue
		}
		key, value, ok := parseEnvDefaultField(line)
		if !ok || current == "" {
			continue
		}
		defaults[current+key] = value
	}

	require.NotEmpty(t, defaults, "envspec.go から envDefault を 1 件も抽出できず、検証が空振りする")
	return defaults
}

// collectEnvPrefixes は、Loader のサブシステムフィールドから型名 → envPrefix の対応を返します。
func collectEnvPrefixes(lines []string) map[string]string {
	prefixes := map[string]string{}
	for _, line := range lines {
		if m := loaderPrefixRe.FindStringSubmatch(line); m != nil {
			prefixes[m[1]] = m[2]
		}
	}
	return prefixes
}

// parseEnvDefaultField は、フィールド宣言行から env キー名と envDefault 値を取り出します。
// envDefault を持たない行と、caarlos0/env が読み飛ばす env:"-" は対象外です。
func parseEnvDefaultField(line string) (string, string, bool) {
	tag := envTagRe.FindStringSubmatch(line)
	def := envDefaultRe.FindStringSubmatch(line)
	if tag == nil || def == nil || tag[1] == "-" {
		return "", "", false
	}
	return tag[1], def[1], true
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
