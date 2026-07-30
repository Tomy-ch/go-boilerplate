package architest

import (
	"path/filepath"
	"regexp"
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

// envReadmeFile は、env 変数一覧の正本（英語）です。日本語版 README.ja.md は正本の翻訳であり、
// 対訳同期は別の手当てに委ねるため検証対象に含めません。
const envReadmeFile = "env/README.md"

// envSpecFile は、env 契約の SSOT である Loader 構造体の宣言元です。
const envSpecFile = "internal/config/envspec.go"

// targetStatusCodesKey は、環境別ポリシーを持つ唯一のキーです。
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

// placeholderMarker は、Example 列が実値ではなくプレースホルダであることを宣言する Notes 列の記法です。
const placeholderMarker = "Example is a placeholder"

var (
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

// envSpecField は、envspec.go が宣言する 1 キーの既定値です。hasDefault が false のキーは
// envDefault タグを持たず、env ファイルへの記載が必須になります。
type envSpecField struct {
	def        string
	hasDefault bool
}

// TestEnvTargetStatusCodesPolicy は、OBS_TARGET_STATUS_CODES の値が env ファイル間で宣言された
// 環境別ポリシーどおりに分岐していることを機械検証します。
// env ファイルは互いに独立した手書きテキストで、キーの値が環境ごとに違ってよいのか揃うべきなのかを
// 表現する場所が無く、伝播漏れと意図的なポリシーが同じ見た目になります。本テストがその区別を担い、
// 「一部の env にだけ足した」も「揃えるべきでない環境を揃えた」も loud な失敗に変えます。
func TestEnvTargetStatusCodesPolicy(t *testing.T) {
	t.Parallel()

	root := moduleRoot(t)

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
					"意図的な変更なら targetStatusCodesPolicy と env/README.md の Notes を更新すること",
				file, targetStatusCodesKey, strings.Join(tier.excluded, ","))
		}
		upper = readStatusCodes(t, root, tier.files[0])
	}
}

// TestEnvReadmeExamples は、env/README.md の Example 列が全キーでローカル実効値と一致し、
// 表のキー集合が envspec.go と 1:1 であることを検証します。
// Example 列は env ファイルと envDefault タグの複製であり、片方だけ更新しても何も検知しません。
// Notes 列がプレースホルダを宣言する行だけは、値の一致ではなく非空を固定します。
func TestEnvReadmeExamples(t *testing.T) {
	t.Parallel()

	root := moduleRoot(t)
	rows := parseEnvReadmeRows(t, root)
	local := readLocalEnv(t, root)
	spec := parseEnvSpec(t, root)

	for key, field := range spec {
		row, ok := rows[key]
		if !assert.Truef(t, ok, "%s は envspec.go が宣言しているが %s に行が無い", key, envReadmeFile) {
			continue
		}

		example := unwrapExample(row.example)
		if strings.Contains(row.notes, placeholderMarker) {
			assert.NotEmptyf(t, example, "%s は Example をプレースホルダと宣言しているが空になっている", key)
			continue
		}

		want, ok := local[key]
		if !ok {
			if !assert.Truef(t, field.hasDefault,
				"%s は %s に無く envDefault も持たないため、Example に載せるローカル実効値が定まらない", key, envLocalFile) {
				continue
			}
			want = field.def
		}
		assert.Equalf(t, want, example, "%s の Example がローカル実効値と一致しない", key)
	}

	for key := range rows {
		_, ok := spec[key]
		assert.Truef(t, ok, "%s は %s に行があるが envspec.go が宣言していない", key, envReadmeFile)
	}
}

// TestEnvReadmeCodeDefaults は、env/README.md の Notes 列にある Code default の記載が
// internal/config/envspec.go の envDefault タグと双方向に一致することを検証します。
// Notes 列の記載はタグの複製であり、既定値を変えて README を直し忘れれば、運用者は存在しない
// 既定値を読むことになります。
func TestEnvReadmeCodeDefaults(t *testing.T) {
	t.Parallel()

	root := moduleRoot(t)
	spec := parseEnvSpec(t, root)
	documented := collectCodeDefaults(t, root)

	for key, field := range spec {
		if !field.hasDefault {
			continue
		}
		got, ok := documented[key]
		if !assert.Truef(t, ok, "%s は envDefault:%q を持つが %s に Code default の記載が無い", key, field.def, envReadmeFile) {
			continue
		}
		assert.Equalf(t, field.def, got, "%s の Code default が envspec.go の envDefault と一致しない", key)
	}

	for key := range documented {
		assert.Truef(t, spec[key].hasDefault,
			"%s は %s に Code default と記載されているが envspec.go に envDefault が無い", key, envReadmeFile)
	}
}

// readStatusCodes は、env ファイルの OBS_TARGET_STATUS_CODES をカンマ区切りで分解して返します。
func readStatusCodes(t *testing.T, root, file string) []string {
	t.Helper()

	value, ok := parseEnvFile(t, root, file)[targetStatusCodesKey]
	require.Truef(t, ok, "%s に %s が無い", file, targetStatusCodesKey)
	return strings.Split(value, ",")
}

// readLocalEnv は、env/README.md の Example 列が記述するローカル既定の env 値を返します。
// CI は埋め込みのため env/.env を env/.env.<APP_ENV> で上書きする（make materialize-env）ので、
// 作業ツリーの APP_ENV がローカルでなければコミット済みの内容を読み直します。
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

// unwrapExample は、Example 列のセルからバッククォート囲みを外した値を返します。
// URL は markdownlint が裸置きを許さないため囲みが要り、囲みの有無は値の一部ではありません。
func unwrapExample(cell string) string {
	cell = strings.TrimSpace(cell)
	if inner, ok := strings.CutPrefix(cell, "`"); ok {
		if inner, ok = strings.CutSuffix(inner, "`"); ok {
			return inner
		}
	}
	return cell
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

// parseEnvSpec は、envspec.go が宣言する env キーと、その envDefault の有無・値を返します。
// キーは Loader のサブシステムフィールドが持つ envPrefix を連ねた完全形にします。
// depguard が go/ast を禁じるため、gofmt 済みソースのテキスト走査で抽出します（既存 architest と同方針）。
func parseEnvSpec(t *testing.T, root string) map[string]envSpecField {
	t.Helper()

	lines := strings.Split(readRepoFile(t, root, envSpecFile), "\n")
	prefixes := collectEnvPrefixes(lines)
	require.NotEmpty(t, prefixes, "Loader から envPrefix を 1 件も抽出できず、検証が空振りする")

	spec := map[string]envSpecField{}
	current := ""
	inSubsystem := false
	for _, line := range lines {
		if m := structDeclRe.FindStringSubmatch(line); m != nil {
			current, inSubsystem = prefixes[m[1]]
			continue
		}
		key, field, ok := parseEnvSpecField(line)
		if !ok || !inSubsystem {
			continue
		}
		spec[current+key] = field
	}

	require.NotEmpty(t, spec, "envspec.go から env キーを 1 件も抽出できず、検証が空振りする")
	return spec
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

// parseEnvSpecField は、フィールド宣言行から env キー名と envDefault の有無・値を取り出します。
// caarlos0/env が読み飛ばす env:"-" は対象外です。
func parseEnvSpecField(line string) (string, envSpecField, bool) {
	tag := envTagRe.FindStringSubmatch(line)
	if tag == nil || tag[1] == "-" {
		return "", envSpecField{}, false
	}
	if def := envDefaultRe.FindStringSubmatch(line); def != nil {
		return tag[1], envSpecField{def: def[1], hasDefault: true}, true
	}
	return tag[1], envSpecField{}, true
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
