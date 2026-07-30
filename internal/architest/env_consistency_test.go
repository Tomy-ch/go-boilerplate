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

	pkgexec "go-boilerplate/pkg/exec"
	pkgfs "go-boilerplate/pkg/fs"

	"github.com/joho/godotenv"
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

// targetStatusCodesKey は、絞り込みの形まで宣言された環境別ポリシーを持つキーです。
const targetStatusCodesKey = "OBS_TARGET_STATUS_CODES"

// perEnvValueMarker は、env ファイル間で値が異なることを宣言する Notes 列の記法です。
const perEnvValueMarker = "Per-environment value"

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

// deployInjectedMarker は、デプロイ基盤が実行時に値を注入するため env ファイルには書かないことを
// 宣言する Notes 列の記法です。
const deployInjectedMarker = "Injected at deploy time"

var (
	// envValueFiles は、値を突き合わせる env ファイルの全件です（ローカル既定と各環境ファイル）。
	envValueFiles = []string{envLocalFile, "env/.env.ci", "env/.env.dev", "env/.env.stg", "env/.env.prd"}
	// envDeployFiles は、デプロイ基盤が実行時に値を注入しうる環境の env ファイルです。ここに挙がらない
	// ファイルはリポジトリ自身が値を持つ環境であり、キーの不在をデプロイ時注入として説明できません。
	envDeployFiles = []string{"env/.env.dev", "env/.env.stg", "env/.env.prd"}
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
	// readmeTagColonRe / readmeTagPhraseRe は、README の散文が参照するタグ名を、2 通りの綴りから捕捉します。
	// コロン付きはタグ名を名乗る綴りが他に無いため文脈を問わず拾い、コロン無しは普通名詞と区別が付かないため
	// 直後に tag と続く言い回しに限ります。前者を文脈非依存にしてあるのは、言い回しが変わったときに参照が
	// 黙って抽出対象から外れる（検証が縮む）のを避けるためです。
	readmeTagColonRe  = regexp.MustCompile("`([A-Za-z]\\w*):`")
	readmeTagPhraseRe = regexp.MustCompile("`([A-Za-z]\\w*)` ?(?:tag|タグ)")
	// structTagRe / structTagKeyRe は、フィールドタグのリテラルと、そこで使われているタグ名を捕捉します。
	structTagRe    = regexp.MustCompile("`[^`]*`")
	structTagKeyRe = regexp.MustCompile(`(\w+):"`)
	// typeVocabularyRe は、Conventions の型対応列挙から Type 列に書ける語を捕捉します。
	typeVocabularyRe = regexp.MustCompile("`([a-z][a-z0-9]*)` ?→")
	// envReadmeFiles は、env 変数一覧の全件です。言語に依らない記載は対訳側も同じ検証に載せられます。
	envReadmeFiles = []string{envReadmeFile, envReadmeTranslationFile}
)

// envTypeVocabulary は、変数表の Type 列に書ける語の宣言です。env/README.md の Conventions が
// 同じ語彙を列挙しており、両者の一致は TestEnvReadmeTypeVocabulary が検証します。
var envTypeVocabulary = []string{"string", "int", "bool", "duration", "csv"}

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

// TestEnvPerEnvironmentValuePolicy は、env ファイル間で実効値が割れるキーの集合と、env/README.md の
// Notes 列に置かれた環境差マーカーの集合が双方向に一致することを機械検証します。
// env ファイルは互いに独立した手書きテキストで、値が環境ごとに違ってよいのか揃うべきなのかを
// 表現する場所が無く、伝播漏れと意図的なポリシーが同じ見た目になります。README の Notes に
// 書かれた宣言だけがその区別を持つため、マーカーの無いキーが割れている状態（伝播漏れ）と、
// 値が揃ったのにマーカーが残った状態（陳腐化した宣言）の双方を loud な失敗に変えます。
// 比較するのは記載された値ではなく実効値です。envDefault を持つキーを 1 つの env ファイルだけで
// 上書きした状態は、記載された値だけを見ると常に 1 種類で、環境差として現れないためです。
func TestEnvPerEnvironmentValuePolicy(t *testing.T) {
	t.Parallel()

	root := moduleRoot(t)
	rows := envReadmeRowsByKey(parseEnvReadmeRows(t, root, envReadmeFile))
	values := readEnvFileValues(t, root)
	spec := parseEnvSpec(t, root)

	for _, key := range slices.Sorted(maps.Keys(values)) {
		row, ok := rows[key]
		if !assert.Truef(t, ok, "%s が env ファイルにあるが %s の変数表に無い", key, envReadmeFile) {
			continue
		}

		split := describeValueSplit(values[key], spec[key])
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

// TestEnvRequiredKeyPresencePolicy は、envDefault を持たないキーが全 env ファイルに記載されている
// ことと、その唯一の例外であるデプロイ時注入が env/README.md の Notes 列で宣言されていることを
// 双方向に機械検証します。
// 値の割れと違い、キーが丸ごと欠落した状態は残ったファイルの値と何も矛盾しません。値の突き合わせは
// 宣言しているファイルしか見ないため欠落は素通りし、その env でアプリを起動して required
// バリデーションが落ちるまで顕在化しません。マーカーの無いキーの欠落（伝播漏れ）と、マーカーの
// 陳腐化（deploy 環境のファイルへ値が戻った、記載を持つ環境から消えた、既定値を持つキーに付けた）の
// 双方を loud な失敗に変えます。
func TestEnvRequiredKeyPresencePolicy(t *testing.T) {
	t.Parallel()

	root := moduleRoot(t)
	rows := envReadmeRowsByKey(parseEnvReadmeRows(t, root, envReadmeFile))
	values := readEnvFileValues(t, root)
	spec := parseEnvSpec(t, root)

	for _, file := range envDeployFiles {
		require.Containsf(t, envValueFiles, file,
			"%s が envValueFiles に無く、不在を許す判定が空振りする", file)
	}

	required := make([]string, 0, len(spec))
	for _, key := range slices.Sorted(maps.Keys(spec)) {
		// 表に行が無いキーは TestEnvReadmeExamples が落とすため、ここでは宣言を読めないものとして飛ばします。
		row, ok := rows[key]
		if !ok {
			continue
		}
		if spec[key].hasDefault {
			assert.Falsef(t, strings.Contains(row.notes, deployInjectedMarker),
				"%s は envDefault を持ち全 env ファイルでの不在が正常なので、%q は宣言として意味を成さない",
				key, deployInjectedMarker)
			continue
		}
		required = append(required, key)
	}
	require.NotEmpty(t, required,
		"envDefault を持たないキーが 1 件も無く、記載の検証が空振りする")

	for _, key := range required {
		injected := strings.Contains(rows[key].notes, deployInjectedMarker)
		for _, file := range envValueFiles {
			_, declared := values[key][file]
			if injected && slices.Contains(envDeployFiles, file) {
				assert.Falsef(t, declared,
					"%s の Notes は %q と宣言しているが %s に記載がある。"+
						"宣言が陳腐化しているのでマーカーを外すこと", key, deployInjectedMarker, file)
				continue
			}
			assert.Truef(t, declared,
				"%s は envDefault を持たないのに %s に記載が無い。伝播漏れならその env の値を書き、"+
					"デプロイ基盤が注入するなら %s の Notes に %q と書くこと",
				key, file, envReadmeFile, deployInjectedMarker)
		}
	}
}

// TestEnvReadmeExamples は、env/README.md の Example 列が全キーでローカル実効値と一致し、
// 表のキー集合が envspec.go と 1:1 であることを検証します。
// Example 列は env ファイルと envDefault タグの複製であり、片方だけ更新しても何も検知しません。
// Notes 列がプレースホルダを宣言する行だけは、値の一致ではなく非空を固定します。
func TestEnvReadmeExamples(t *testing.T) {
	t.Parallel()

	root := moduleRoot(t)
	rows := envReadmeRowsByKey(parseEnvReadmeRows(t, root, envReadmeFile))
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
	documented := collectCodeDefaults(t, root, envReadmeFile)

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

// TestEnvReadmeTranslationValues は、対訳の変数表が正本と同じ値を載せていることを検証します。
// 対訳は正本の全内容を複製しており、値も同じだけ載っています。このリポジトリの読者は日本語版を読む
// 前提なので、正本にだけ検証を入れると、誤った既定値を読む確率はむしろ対訳側の方が高くなります。
// 突き合わせるのはキー / Type / Example / Code default / 検証指定に限ります。Description 列と
// Notes 列の散文は翻訳で分岐するため、一致を求めれば翻訳そのものを禁じることになります。
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

// TestEnvReadmeTagNames は、env 変数一覧の散文が参照する struct タグ名が
// internal/config/envspec.go に実在するタグ名であることを、正本と対訳の両方について検証します。
// 散文中のタグ名はコードの識別子の手書きの複製で、実在しない名前を書いても Markdown は通り、
// 既定値の一致を見る TestEnvReadmeCodeDefaults もタグ名そのものは読みません。誤った名前のまま
// 手順どおりに変数を追加すると、caarlos0/env は未知のタグを黙って読み飛ばし、既定値が効かないまま
// required でもないフィールドが残ります。対訳も同じ検証に載せられるのは、翻訳で綴りが分かれる目印と
// 違い、タグ名がコードの識別子そのもので訳されないためです。
func TestEnvReadmeTagNames(t *testing.T) {
	t.Parallel()

	root := moduleRoot(t)
	declared := collectStructTagKeys(t, root)

	for _, file := range envReadmeFiles {
		referenced := collectReadmeTagRefs(t, root, file)
		require.NotEmptyf(t, referenced, "%s からタグ名の参照を 1 件も抽出できず、検証が空振りする", file)

		for _, name := range referenced {
			assert.Containsf(t, declared, name,
				"%s が参照するタグ名 %s を %s は使っていない", file, name, envSpecFile)
		}
	}
}

// TestEnvReadmeTypeVocabulary は、変数表の Type 列が Conventions の定める語彙だけで構成されて
// いること、およびその語彙が正本・対訳・テスト宣言の三者で一致することを検証します。
// 見るのは語彙に属するかどうかだけで、その語が当のキーの Go フィールド型と対応しているかまでは
// 見ません。語彙外の値は、未定義の型を書いたか、Markdown のセル分割が崩れて別の列を Type として
// 読んでいるかのどちらかで、後者は表の見た目では気づけません。語彙の宣言を README とテストの双方に
// 置くのは、語彙を増やす変更に両方を触らせ、README だけを広げて検証が黙って緩むのを防ぐためです。
// 対訳の列挙まで見るのは、語彙が Conventions 自身の定める固定のトークンであって訳す対象ではなく、
// 対訳にも正本と同じ綴りで並ぶためです。
func TestEnvReadmeTypeVocabulary(t *testing.T) {
	t.Parallel()

	root := moduleRoot(t)

	for _, file := range envReadmeFiles {
		assert.ElementsMatchf(t, envTypeVocabulary, collectTypeVocabulary(t, root, file),
			"%s の Conventions が列挙する Type 語彙が envTypeVocabulary と一致しない。"+
				"型を増やしたなら両方を更新すること", file)
	}

	for _, row := range parseEnvReadmeRows(t, root, envReadmeFile) {
		assert.Containsf(t, envTypeVocabulary, strings.TrimSpace(row.typ),
			"%s の %s の Type 列が Conventions の語彙に無い。未定義の型か、セルがずれて別の列を"+
				"Type として読んでいる", envReadmeFile, row.key)
	}
}

// collectTypeVocabulary は、Conventions の型対応列挙が挙げる Type 列の語を重複を除いて返します。
func collectTypeVocabulary(t *testing.T, root, file string) []string {
	t.Helper()

	names := []string{}
	for _, m := range typeVocabularyRe.FindAllStringSubmatch(readRepoFile(t, root, file), -1) {
		if !slices.Contains(names, m[1]) {
			names = append(names, m[1])
		}
	}

	require.NotEmptyf(t, names, "%s から Type 語彙を 1 件も抽出できず、検証が空振りする", file)
	return names
}

// collectReadmeTagRefs は、env 変数一覧の散文が参照するタグ名の全件を重複を除いて返します。
func collectReadmeTagRefs(t *testing.T, root, file string) []string {
	t.Helper()

	readme := readRepoFile(t, root, file)

	names := []string{}
	for _, re := range []*regexp.Regexp{readmeTagColonRe, readmeTagPhraseRe} {
		for _, m := range re.FindAllStringSubmatch(readme, -1) {
			if !slices.Contains(names, m[1]) {
				names = append(names, m[1])
			}
		}
	}
	return names
}

// collectStructTagKeys は、envspec.go のフィールドタグが使っているタグ名の全件を返します。
func collectStructTagKeys(t *testing.T, root string) []string {
	t.Helper()

	keys := []string{}
	for _, tag := range structTagRe.FindAllString(readRepoFile(t, root, envSpecFile), -1) {
		for _, m := range structTagKeyRe.FindAllStringSubmatch(tag, -1) {
			if !slices.Contains(keys, m[1]) {
				keys = append(keys, m[1])
			}
		}
	}

	require.NotEmptyf(t, keys, "%s からタグ名を 1 件も抽出できず、検証が空振りする", envSpecFile)
	return keys
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

// describeValueSplit は、1 つのキーの実効値が env ファイル間で割れているときにその内訳を返します。
// envDefault を持つキーは、記載の無いファイルを既定値で補完して比較します。記載が無くても実効値は
// 既定値で確定するため、1 つの env ファイルだけが既定から外れている状態も割れとして現れます。
// envDefault を持たないキーの不在は、その環境の値が外部から注入されて未知であり値の差ではないため、
// 比較に含めません（不在そのものは TestEnvRequiredKeyPresencePolicy が見ます）。
// 揃っていれば nil を返します。
func describeValueSplit(byFile map[string]string, field envSpecField) []string {
	effective := effectiveValues(byFile, field)

	distinct := map[string]struct{}{}
	for _, value := range effective {
		distinct[value] = struct{}{}
	}
	if len(distinct) <= 1 {
		return nil
	}

	split := make([]string, 0, len(effective))
	for _, file := range envValueFiles {
		value, ok := effective[file]
		if !ok {
			continue
		}
		if _, declared := byFile[file]; !declared {
			split = append(split, fmt.Sprintf("%s=%q(envDefault)", file, value))
			continue
		}
		split = append(split, fmt.Sprintf("%s=%q", file, value))
	}
	return split
}

// effectiveValues は、1 つのキーが env ファイルごとに持つ実効値を返します。記載の無いファイルは、
// envDefault があればその値を、無ければ実効値が定まらないものとして落とします。
func effectiveValues(byFile map[string]string, field envSpecField) map[string]string {
	effective := make(map[string]string, len(envValueFiles))
	for _, file := range envValueFiles {
		value, declared := byFile[file]
		if declared {
			effective[file] = value
			continue
		}
		if field.hasDefault {
			effective[file] = field.def
		}
	}
	return effective
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

// envReadmeRowsByKey は、変数表の行をキーで引ける形に並べ替えて返します。
func envReadmeRowsByKey(rows []readmeRow) map[string]readmeRow {
	byKey := make(map[string]readmeRow, len(rows))
	for _, row := range rows {
		byKey[row.key] = row
	}
	return byKey
}

// envReadmeKeys は、変数表の行から記載順のキー列を返します。
func envReadmeKeys(rows []readmeRow) []string {
	keys := make([]string, 0, len(rows))
	for _, row := range rows {
		keys = append(keys, row.key)
	}
	return keys
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
