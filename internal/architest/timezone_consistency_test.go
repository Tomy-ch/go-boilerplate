package architest

import (
	"maps"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	pkgfs "go-boilerplate/pkg/fs"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// osTimeZoneKey は、タイムゾーンの正本である env キーです。
const osTimeZoneKey = "OS_TZ"

// serverDockerfile は、アプリイメージのローカルタイムを決める ENV TZ の宣言元です。
const serverDockerfile = "docker/server/Dockerfile"

// unnamedDockerfileStage は、AS を持たない FROM に与える表示名です。
const unnamedDockerfileStage = "(名前なし)"

var (
	// composeFilePattern / workflowFilePatterns は、TZ を宣言しうる YAML の全件を列挙する glob です。
	// 変数名ではなくファイルの所在で列挙するのは、変数名で grep すると宣言済みのファイルしか
	// 見つからず、まだ宣言していないファイルだけを黙って取りこぼすためです。
	composeFilePattern   = "docker-compose*.yaml"
	workflowFilePatterns = []string{
		filepath.Join(".github", "workflows", "*.yaml"),
		filepath.Join(".github", "workflows", "*.yml"),
	}
	// yamlTimezoneRe は、YAML 中の TZ: / PGTZ: の行から名前と値を捕捉します。environment ブロック
	// 配下に置かれる想定ですが、インデントの階層は見ていないため所属の検証はしません。
	yamlTimezoneRe = regexp.MustCompile(`(?m)^\s*(TZ|PGTZ):\s*(\S+)\s*$`)
	// postgresImageRe は、PostgreSQL を起動するサービス定義を image 名から判定します。
	postgresImageRe = regexp.MustCompile(`(?m)^\s*image:\s*postgres[:@]`)
	// dockerfileFromRe は、ステージ境界となる FROM 行からステージ名を捕捉します。イメージ名の前に
	// 任意個の --platform 等のフラグを許すのは、フラグ付きの FROM を境界として認識し損ねると、
	// 以降の宣言が直前のステージへ誤って帰属して検証が静かに歪むためです。
	dockerfileFromRe = regexp.MustCompile(`^FROM\s+(?:--\S+\s+)*\S+(?:\s+(?i:AS)\s+(\S+))?$`)
	// dockerfileEnvRe / dockerfileEnvTZRe は、ENV 行の代入部を取り出し、そこから TZ の値を捕捉します。
	// ENV は 1 行に複数の変数を並べられるため、行全体ではなく代入トークン単位で見ます。
	dockerfileEnvRe   = regexp.MustCompile(`^ENV\s+(.+)$`)
	dockerfileEnvTZRe = regexp.MustCompile(`(?:^|\s)TZ[=\s]+(\S+)`)
	// dockerfileTzdataRe は、ステージが tzdata パッケージを導入する行を検知します。
	dockerfileTzdataRe = regexp.MustCompile(`\btzdata\b`)
	// dockerfileContinuationRe は、次行へ継続する行の末尾の \ を捕捉します。
	dockerfileContinuationRe = regexp.MustCompile(`\\\s*$`)
)

// dockerfileStage は、Dockerfile の 1 ステージから取り出した、ローカルタイムに関わる宣言です。
type dockerfileStage struct {
	name           string
	timeZone       string
	hasTimeZone    bool
	installsTzdata bool
}

// TestTimezoneMechanismValuesMatch は、タイムゾーンを供給する全機構が同じ値を宣言していることを
// 機械検証します。
//
// 機構は env ファイル / compose / ワークフロー / Dockerfile に分散した独立の手書きテキストで、
// 互いを参照しません。片方だけ変えても各ファイルは単体で妥当なままなので、伝播漏れは
// 実際に時刻を読むまで表面化せず、しかもその時刻は「9 時間ずれた妥当な値」として返ります。
// env/README.md の Changing the Timezone は触る箇所を列挙していますが、列挙は読まれなければ
// 効かないため、本テストが読み落としを loud な失敗に変えます。
func TestTimezoneMechanismValuesMatch(t *testing.T) {
	t.Parallel()

	root := moduleRoot(t)

	byFile := readEnvFileValues(t, root)[osTimeZoneKey]
	require.NotEmptyf(t, byFile, "%s がどの env ファイルにも無く、比較の基準が定まらない", osTimeZoneKey)
	want := byFile[envLocalFile]
	require.NotEmptyf(t, want, "%s に %s が無く、比較の基準が定まらない", envLocalFile, osTimeZoneKey)

	// 単一の値を全機構へ配る前提そのものを固定します。env ファイル間で値が割れると、
	// 単一の ENV TZ と突き合わせる以降の比較は基準を失うため、先にここで揃いを確かめます。
	for _, file := range slices.Sorted(maps.Keys(byFile)) {
		assert.Equalf(t, want, byFile[file],
			"%s の %s が %s と一致しない。単一の値を全機構へ配る前提が崩れるため、"+
				"環境別に倒すなら本テストと env/README.md の Changing the Timezone を先に更新すること",
			file, osTimeZoneKey, envLocalFile)
	}

	files := append(globRepoFiles(t, root, composeFilePattern), globRepoFiles(t, root, workflowFilePatterns...)...)
	declarations := 0
	for _, file := range files {
		for _, decl := range yamlTimezoneRe.FindAllStringSubmatch(readRepoFile(t, root, file), -1) {
			declarations++
			assert.Equalf(t, want, decl[2],
				"%s の %s が %s の %s と一致しない。env/README.md の Changing the Timezone を参照して全機構を揃えること",
				file, decl[1], envLocalFile, osTimeZoneKey)
		}
	}
	require.NotZerof(t, declarations, "compose / ワークフローから TZ の宣言を 1 件も抽出できず、検証が空振りする")

	declared := 0
	for _, stage := range readDockerfileStages(t, root, serverDockerfile) {
		if !stage.hasTimeZone {
			continue
		}
		declared++
		assert.Equalf(t, want, stage.timeZone,
			"%s の %s ステージの ENV TZ が %s の %s と一致しない",
			serverDockerfile, stage.name, envLocalFile, osTimeZoneKey)
	}
	require.NotZerof(t, declared, "%s から ENV TZ を 1 件も抽出できず、検証が空振りする", serverDockerfile)
}

// TestDockerfileTzdataStagesDeclareTimeZone は、tzdata を導入した全ステージが ENV TZ も宣言して
// いることを機械検証します。
//
// Go は TZ も /etc/localtime も無いプロセスでは time.Local を UTC にします。tzdata だけを入れた
// ステージはビルドも起動も成功し、コンテナ内で date を叩くまで誰も気付かないまま設定と無関係な
// UTC で動きます。tzdata の導入は「このステージはローカルタイムを持つ」という意思表示なので、
// その対で ENV TZ を要求します。
func TestDockerfileTzdataStagesDeclareTimeZone(t *testing.T) {
	t.Parallel()

	root := moduleRoot(t)

	withTzdata := 0
	for _, stage := range readDockerfileStages(t, root, serverDockerfile) {
		if !stage.installsTzdata {
			continue
		}
		withTzdata++
		assert.Truef(t, stage.hasTimeZone,
			"%s の %s ステージは tzdata を導入しているが ENV TZ が無い。"+
				"Go の time.Local は UTC へ黙って落ちるため、ENV TZ を宣言すること",
			serverDockerfile, stage.name)
	}
	require.NotZerof(t, withTzdata,
		"%s から tzdata を導入するステージを 1 件も抽出できず、検証が空振りする", serverDockerfile)
}

// TestPostgresProvisionersDeclareTimeZone は、PostgreSQL を起動する compose / ワークフローが
// TZ と PGTZ を宣言していることを機械検証します。
//
// 値の一致を見る TestTimezoneMechanismValuesMatch は、宣言が丸ごと消えた欠落を検出できません
// （他のファイルに同じキーが残る限り、比較対象が減るだけで一致は保たれます）。宣言の存在を
// 要求するのはこのテストだけで、compose とワークフローの両方を対象にするのはそのためです。
//
// GitHub Actions はサービス定義をワークフロー間で共有できないため、同じ宣言が全ファイルに写経
// されます。写経の漏れは変数名での grep には映らず（宣言済みのファイルしか引っ掛からない）、
// 漏れたファイルだけが UTC のクラスタ既定で走ります。判定を image 名側から行うことで、
// 「PostgreSQL を使うのに TZ を宣言していない」ファイルを取りこぼしません。
//
// 宣言の有無はファイル全体で見ており、当該サービスのブロックへは紐付けていません。1 ファイルに
// PostgreSQL 以外のサービスが増え、そちらだけが TZ を宣言する状態は素通りします。YAML の階層を
// テキスト走査で切り出す脆さを負うより、1 ファイル 1 PostgreSQL という現状に合わせた割り切りです。
func TestPostgresProvisionersDeclareTimeZone(t *testing.T) {
	t.Parallel()

	root := moduleRoot(t)

	files := append(globRepoFiles(t, root, composeFilePattern), globRepoFiles(t, root, workflowFilePatterns...)...)
	provisioning := 0
	for _, file := range files {
		content := readRepoFile(t, root, file)
		if !postgresImageRe.MatchString(content) {
			continue
		}
		provisioning++

		var declared []string
		for _, decl := range yamlTimezoneRe.FindAllStringSubmatch(content, -1) {
			declared = append(declared, decl[1])
		}
		assert.Containsf(t, declared, "TZ",
			"%s は PostgreSQL を起動するが TZ を宣言していない。initdb のクラスタ既定が UTC になる", file)
		assert.Containsf(t, declared, "PGTZ",
			"%s は PostgreSQL を起動するが PGTZ を宣言していない。psql セッションが UTC で表示される", file)
	}
	require.NotZerof(t, provisioning,
		"PostgreSQL を起動する compose / ワークフローを 1 件も抽出できず、検証が空振りする")
}

// Test_parseDockerfileStages は、Dockerfile の書き方の揺れに対して、ステージの切り出しと
// ローカルタイムに関わる宣言の帰属が保たれることを検証します。
//
// このパーサが取りこぼすと、宣言が直前のステージへ帰属したり、宣言済みの TZ を未宣言と見なした
// りして、上の 3 本が静かに歪みます。実リポジトリの Dockerfile は現在ひととおりの書き方しか
// 使っていないため、揺れは合成入力でしか固定できません。
func Test_parseDockerfileStages(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("FROM ごとにステージを切り出し宣言をそのステージへ帰属させる", func(t *testing.T) {
			t.Parallel()

			got := parseDockerfileStages("FROM alpine AS runtime\nRUN apk add tzdata\nENV TZ=Asia/Tokyo\n" +
				"FROM alpine AS other\nENV FOO=bar\n")

			require.Len(t, got, 2)
			assert.Equal(t, dockerfileStage{name: "runtime", timeZone: "Asia/Tokyo", hasTimeZone: true, installsTzdata: true}, got[0])
			assert.Equal(t, dockerfileStage{name: "other"}, got[1])
		})

		t.Run("イメージ名の前にフラグを持つ FROM もステージ境界として扱う", func(t *testing.T) {
			t.Parallel()

			got := parseDockerfileStages("FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder\n" +
				"FROM alpine AS runtime\nENV TZ=Asia/Tokyo\n")

			require.Len(t, got, 2)
			assert.Equal(t, "builder", got[0].name)
			assert.False(t, got[0].hasTimeZone, "TZ は後続ステージの宣言であり builder へ帰属してはならない")
			assert.Equal(t, "Asia/Tokyo", got[1].timeZone)
		})

		t.Run("1 行に複数の変数を並べた ENV からも TZ を捕捉する", func(t *testing.T) {
			t.Parallel()

			got := parseDockerfileStages("FROM alpine AS runtime\nENV CGO_ENABLED=0 TZ=Asia/Tokyo GOFLAGS=-mod=vendor\n")

			require.Len(t, got, 1)
			assert.Equal(t, "Asia/Tokyo", got[0].timeZone)
		})

		t.Run("継続行にまたがる ENV と RUN からも宣言を捕捉する", func(t *testing.T) {
			t.Parallel()

			got := parseDockerfileStages("FROM alpine AS runtime\nRUN apk add --no-cache \\\n    ca-certificates \\\n    tzdata\n" +
				"ENV GOPROXY=off \\\n    TZ=Asia/Tokyo\n")

			require.Len(t, got, 1)
			assert.True(t, got[0].installsTzdata)
			assert.Equal(t, "Asia/Tokyo", got[0].timeZone)
		})

		t.Run("AS を持たない FROM も 1 ステージとして数える", func(t *testing.T) {
			t.Parallel()

			got := parseDockerfileStages("FROM alpine\nRUN apk add tzdata\n")

			require.Len(t, got, 1)
			assert.Equal(t, unnamedDockerfileStage, got[0].name)
			assert.True(t, got[0].installsTzdata)
			assert.False(t, got[0].hasTimeZone)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("FROM の前に置かれた宣言はどのステージにも帰属しない", func(t *testing.T) {
			t.Parallel()

			got := parseDockerfileStages("ENV TZ=Asia/Tokyo\nFROM alpine AS runtime\n")

			require.Len(t, got, 1)
			assert.False(t, got[0].hasTimeZone, "FROM より前の宣言はどのステージのものでもない")
		})

		t.Run("FROM を含まない内容からはステージを 1 件も返さない", func(t *testing.T) {
			t.Parallel()

			assert.Empty(t, parseDockerfileStages("RUN apk add tzdata\nENV TZ=Asia/Tokyo\n"))
		})
	})
}

// readDockerfileStages は、Dockerfile を読み、ローカルタイムに関わる宣言をステージ単位で返します。
func readDockerfileStages(t *testing.T, root, file string) []dockerfileStage {
	t.Helper()

	stages := parseDockerfileStages(readRepoFile(t, root, file))
	require.NotEmptyf(t, stages, "%s からステージを 1 件も抽出できず、検証が空振りする", file)
	return stages
}

// parseDockerfileStages は、Dockerfile の内容を FROM 単位のステージへ分け、ローカルタイムに
// 関わる宣言を取り出して返します。継続行は前の行へ畳んでから解釈します。
// depguard が go/ast を禁じるためテキスト走査で行います（既存 architest と同方針）。
func parseDockerfileStages(content string) []dockerfileStage {
	var stages []dockerfileStage
	for _, line := range joinDockerfileContinuations(content) {
		if from := dockerfileFromRe.FindStringSubmatch(line); from != nil {
			name := from[1]
			if name == "" {
				name = unnamedDockerfileStage
			}
			stages = append(stages, dockerfileStage{name: name})
			continue
		}
		if len(stages) == 0 {
			continue
		}

		current := &stages[len(stages)-1]
		if env := dockerfileEnvRe.FindStringSubmatch(line); env != nil {
			if tz := dockerfileEnvTZRe.FindStringSubmatch(env[1]); tz != nil {
				current.timeZone = tz[1]
				current.hasTimeZone = true
			}
		}
		if dockerfileTzdataRe.MatchString(line) {
			current.installsTzdata = true
		}
	}

	return stages
}

// joinDockerfileContinuations は、末尾を \ で継続する行を後続行と 1 行へ畳み、前後の空白を
// 落とした行の並びを返します。複数行に散った ENV や RUN の記述を 1 行として解釈するためです。
func joinDockerfileContinuations(content string) []string {
	var (
		lines   []string
		pending string
	)
	for raw := range strings.SplitSeq(content, "\n") {
		line := strings.TrimSpace(raw)
		if dockerfileContinuationRe.MatchString(line) {
			pending += strings.TrimSpace(dockerfileContinuationRe.ReplaceAllString(line, "")) + " "
			continue
		}
		lines = append(lines, strings.TrimSpace(pending+line))
		pending = ""
	}
	if pending != "" {
		lines = append(lines, strings.TrimSpace(pending))
	}

	return lines
}

// globRepoFiles は、リポジトリルートからの相対 glob に一致するファイルを、同じく相対パスで返します。
func globRepoFiles(t *testing.T, root string, patterns ...string) []string {
	t.Helper()

	var files []string
	for _, pattern := range patterns {
		matched, err := pkgfs.OS{}.Glob(filepath.Join(root, pattern))
		require.NoErrorf(t, err, "%s に一致するファイルを列挙できない", pattern)

		for _, path := range matched {
			rel, err := filepath.Rel(root, path)
			require.NoErrorf(t, err, "%s の相対パスを求められない", path)
			files = append(files, filepath.ToSlash(rel))
		}
	}

	slices.Sort(files)
	return files
}
