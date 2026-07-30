package architest

import (
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"testing"

	pkgfs "go-boilerplate/pkg/fs"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// envReadmeFile は、env 変数一覧の正本（英語）です。
const envReadmeFile = "env/README.md"

// envReadmeTranslationFile は、正本の対訳です。散文は翻訳されますが、変数表のキー / Type 列 /
// Example 列 / Notes 列の Code default は言語に依らない値であり、正本と一致していなければなりません。
const envReadmeTranslationFile = "env/README.ja.md"

// envSpecFile は、env 契約の SSOT である Loader 構造体の宣言元です。
const envSpecFile = "internal/config/envspec.go"

// targetStatusCodesKey は、環境別ポリシーを持つ唯一のキーです。
const targetStatusCodesKey = "OBS_TARGET_STATUS_CODES"

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
	// subsystemHeadingRe は、変数表を区切るサブシステム見出しから名前を捕捉します。
	subsystemHeadingRe = regexp.MustCompile(`^### (.+)$`)
	// sectionHeadingRe / listItemRe は、節と箇条書き項目を数えるために行頭の記法を判定します。
	sectionHeadingRe = regexp.MustCompile(`^## `)
	listItemRe       = regexp.MustCompile(`^\s*(?:- |[0-9]+\. )`)
	// codeDefaultRe は、Notes 列の Code default 記法からバッククォートで囲まれた値を捕捉します。
	codeDefaultRe = regexp.MustCompile("Code default `([^`]*)`")
	// notesValidationRe は、Notes 列から caarlos0/env の検証指定を捕捉します。
	notesValidationRe = regexp.MustCompile("`(required[^`]*)`")
)

// codeDefaultEmptyNotations は、Code default が空であることを表す綴りをファイルごとに宣言します。
// 空値はバッククォート記法で書けないため散文で綴られ、綴り自体が翻訳で分岐します。言語ごとに綴りを
// 1 つだけ許可することで、抽出後の値は言語に依らず空文字へ揃います。
var codeDefaultEmptyNotations = map[string]string{
	envReadmeFile:            "Code default empty",
	envReadmeTranslationFile: "Code default は空",
}

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

// readmeRow は、env 変数表の 1 行から取り出したセルです。Description 列は翻訳で分岐するため持ちません。
type readmeRow struct {
	key     string
	typ     string
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

// TestEnvReadmeTargetStatusCodesExample は、env/README.md の Example 列が env/.env の実値と
// 一致することを検証します。Example 列はローカル既定を載せる規約ですが、実体は env ファイルの
// 複製であり、片方だけ更新しても何も検知しません。
func TestEnvReadmeTargetStatusCodesExample(t *testing.T) {
	t.Parallel()

	root := moduleRoot(t)

	row, ok := findEnvReadmeRow(parseEnvReadmeRows(t, root, envReadmeFile), targetStatusCodesKey)
	require.Truef(t, ok, "%s の行が %s に無い", targetStatusCodesKey, envReadmeFile)
	assert.Equalf(t, strings.Join(readStatusCodes(t, root, "env/.env"), ","), row.example,
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
	documented := collectCodeDefaults(t, root, envReadmeFile)

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

// TestEnvReadmeTranslationValues は、対訳の変数表が正本と同じ値を載せていることを検証します。
// 対訳は正本の全内容を複製しており、値も同じだけ載っています。このリポジトリの読者は日本語版を読む
// 前提なので、正本にだけ検証を入れると、誤った既定値を読む確率はむしろ対訳側の方が高くなります。
// 突き合わせるのはキー / Type / Example / Code default に限ります。Description 列と Notes 列の
// 散文は翻訳で分岐するため、一致を求めれば翻訳そのものを禁じることになります。
func TestEnvReadmeTranslationValues(t *testing.T) {
	t.Parallel()

	root := moduleRoot(t)
	canonical := parseEnvReadmeRows(t, root, envReadmeFile)
	translated := parseEnvReadmeRows(t, root, envReadmeTranslationFile)

	require.Equalf(t, envReadmeKeys(canonical), envReadmeKeys(translated),
		"変数表のキーが %s と一致しない。行の増減や並べ替えは対訳側だけで起きうる", envReadmeTranslationFile)

	validations := 0
	for i, want := range canonical {
		got := translated[i]
		assert.Equalf(t, want.typ, got.typ, "%s の Type 列が %s と一致しない", want.key, envReadmeFile)
		assert.Equalf(t, want.example, got.example, "%s の Example 列が %s と一致しない", want.key, envReadmeFile)

		wantValidations := notesValidations(want.notes)
		validations += len(wantValidations)
		assert.Equalf(t, wantValidations, notesValidations(got.notes),
			"%s の Notes 列の検証指定が %s と一致しない", want.key, envReadmeFile)
	}
	require.NotEmptyf(t, validations,
		"%s から Notes 列の検証指定を 1 件も抽出できず、その突き合わせが空振りする", envReadmeFile)

	assert.Equalf(t, collectCodeDefaults(t, root, envReadmeFile), collectCodeDefaults(t, root, envReadmeTranslationFile),
		"Notes 列の Code default が %s と一致しない", envReadmeFile)
}

// TestEnvReadmeTranslationStructure は、正本の文書構造が対訳でも保たれていることを検証します。
// 値の突き合わせは表に載った行しか見ないため、節ごと・項目ごと訳し漏らした場合は差として現れません。
// サブシステム見出しは区分そのもので、区切り位置が違えば対訳の読者は変数が属するサブシステムを取り違えます。
// 節と箇条書きは訳文で綴りが変わるため、一致を求められるのは個数だけです。
func TestEnvReadmeTranslationStructure(t *testing.T) {
	t.Parallel()

	root := moduleRoot(t)
	canonical := readRepoFile(t, root, envReadmeFile)
	translated := readRepoFile(t, root, envReadmeTranslationFile)

	assert.Equalf(t, collectSubsystemHeadings(t, root, envReadmeFile), collectSubsystemHeadings(t, root, envReadmeTranslationFile),
		"サブシステム見出しが %s と一致しない", envReadmeFile)
	assert.Equalf(t, countLines(canonical, sectionHeadingRe), countLines(translated, sectionHeadingRe),
		"節の数が %s と一致しない", envReadmeFile)
	assert.Equalf(t, countLines(canonical, listItemRe), countLines(translated, listItemRe),
		"箇条書き項目の数が %s と一致しない。項目ごと訳し漏らしている可能性がある", envReadmeFile)
}

// readStatusCodes は、env ファイルの OBS_TARGET_STATUS_CODES をカンマ区切りで分解して返します。
func readStatusCodes(t *testing.T, root, file string) []string {
	t.Helper()

	for line := range strings.Lines(readRepoFile(t, root, file)) {
		value, ok := strings.CutPrefix(strings.TrimSpace(line), targetStatusCodesKey+"=")
		if !ok {
			continue
		}
		return strings.Split(value, ",")
	}

	require.FailNowf(t, "env ファイルにキーが無い", "%s に %s が無い", file, targetStatusCodesKey)
	return nil
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

// parseEnvReadmeRows は、env 変数表を記載順のまま行へ分解して返します。記載順を保つのは、対訳との
// 突き合わせで行の並べ替えも差として現れるようにするためです。
func parseEnvReadmeRows(t *testing.T, root, file string) []readmeRow {
	t.Helper()

	var rows []readmeRow
	for line := range strings.Lines(readRepoFile(t, root, file)) {
		line = strings.TrimRight(line, "\n")
		m := readmeRowRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		// 行頭と行末の区切りにより前後へ空要素が付くため、Type は 4 番目、Example は 5 番目、
		// Notes は末尾の空要素を除く残り。
		cells := strings.Split(line, "|")
		require.GreaterOrEqualf(t, len(cells), 7, "%s の行の列数が想定と異なる: %s", file, line)
		rows = append(rows, readmeRow{
			key:     m[1],
			typ:     cells[3],
			example: cells[4],
			notes:   strings.Join(cells[5:len(cells)-1], "|"),
		})
	}

	require.NotEmptyf(t, rows, "%s から変数表の行を 1 件も抽出できず、検証が空振りする", file)
	return rows
}

// envReadmeKeys は、変数表の行から記載順のキー列を返します。
func envReadmeKeys(rows []readmeRow) []string {
	keys := make([]string, 0, len(rows))
	for _, row := range rows {
		keys = append(keys, row.key)
	}
	return keys
}

// findEnvReadmeRow は、変数表からキーに対応する行を返します。
func findEnvReadmeRow(rows []readmeRow, key string) (readmeRow, bool) {
	i := slices.IndexFunc(rows, func(row readmeRow) bool { return row.key == key })
	if i < 0 {
		return readmeRow{}, false
	}
	return rows[i], true
}

// collectCodeDefaults は、Notes 列に Code default を持つキーとその値を返します。値が空であることは
// バッククォート記法で書けないため、ファイルごとに宣言した綴りを別途受け付けます。
func collectCodeDefaults(t *testing.T, root, file string) map[string]string {
	t.Helper()

	empty, ok := codeDefaultEmptyNotations[file]
	require.Truef(t, ok, "%s の Code default 空値の綴りが宣言されていない", file)

	defaults := map[string]string{}
	for _, row := range parseEnvReadmeRows(t, root, file) {
		if !strings.Contains(row.notes, "Code default") {
			continue
		}
		if m := codeDefaultRe.FindStringSubmatch(row.notes); m != nil {
			defaults[row.key] = m[1]
			continue
		}
		require.Containsf(t, row.notes, empty,
			"%s の %s の Code default が既定の記法（バッククォート囲み もしくは %q）で書かれていない", file, row.key, empty)
		defaults[row.key] = ""
	}

	require.NotEmptyf(t, defaults, "%s から Code default を 1 件も抽出できず、検証が空振りする", file)
	return defaults
}

// notesValidations は、Notes 列に書かれた env タグの検証指定を記載順に返します。`required` /
// `required,notEmpty` はタグの内容そのもので、周囲の散文と違い翻訳されません。
func notesValidations(notes string) []string {
	matches := notesValidationRe.FindAllStringSubmatch(notes, -1)
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		out = append(out, m[1])
	}
	return out
}

// countLines は、re にマッチする行の数を返します。
func countLines(content string, re *regexp.Regexp) int {
	count := 0
	for line := range strings.Lines(content) {
		if re.MatchString(line) {
			count++
		}
	}
	return count
}

// collectSubsystemHeadings は、変数表を区切るサブシステム見出しを記載順に返します。
func collectSubsystemHeadings(t *testing.T, root, file string) []string {
	t.Helper()

	var headings []string
	for line := range strings.Lines(readRepoFile(t, root, file)) {
		if m := subsystemHeadingRe.FindStringSubmatch(strings.TrimRight(line, "\n")); m != nil {
			headings = append(headings, m[1])
		}
	}

	require.NotEmptyf(t, headings, "%s からサブシステム見出しを 1 件も抽出できず、検証が空振りする", file)
	return headings
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
