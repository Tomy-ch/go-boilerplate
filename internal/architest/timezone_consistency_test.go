package architest

import (
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

var (
	// composeFilePattern / workflowFilePatterns は、TZ を宣言しうる YAML の全件を列挙する glob です。
	// 変数名ではなくファイルの所在で列挙するのは、env/README.md が checklist 3 で述べているとおり、
	// 変数名で grep すると「まだ宣言していないファイル」だけが黙って抜けるためです。
	composeFilePattern   = "docker-compose*.yaml"
	workflowFilePatterns = []string{
		filepath.Join(".github", "workflows", "*.yaml"),
		filepath.Join(".github", "workflows", "*.yml"),
	}
	// yamlTimezoneRe は、YAML の environment 直下に置かれた TZ / PGTZ の宣言から名前と値を捕捉します。
	yamlTimezoneRe = regexp.MustCompile(`(?m)^\s*(TZ|PGTZ):\s*(\S+)\s*$`)
	// postgresImageRe は、PostgreSQL を起動するサービス定義を image 名から判定します。
	postgresImageRe = regexp.MustCompile(`(?m)^\s*image:\s*postgres[:@]`)
	// dockerfileFromRe / dockerfileEnvTZRe は、Dockerfile をステージへ分け、その ENV TZ を捕捉します。
	dockerfileFromRe   = regexp.MustCompile(`^FROM\s+\S+(?:\s+AS\s+(\S+))?\s*$`)
	dockerfileEnvTZRe  = regexp.MustCompile(`^ENV TZ=(\S+)$`)
	dockerfileTzdataRe = regexp.MustCompile(`\btzdata\b`)
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

	want, ok := parseEnvFile(t, root, envLocalFile)[osTimeZoneKey]
	require.Truef(t, ok, "%s に %s が無く、比較の基準が定まらない", envLocalFile, osTimeZoneKey)

	// env ファイル間の一致は TestEnvPerEnvironmentValuePolicy が別途固定していますが、
	// 基準そのものが割れていれば以降の比較の意味が変わるため、ここでも確かめます。
	for _, file := range envValueFiles {
		assert.Equalf(t, want, parseEnvFile(t, root, file)[osTimeZoneKey],
			"%s の %s が %s と一致しない", file, osTimeZoneKey, envLocalFile)
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

	stages := collectDockerfileStages(t, root, serverDockerfile)
	declared := 0
	for _, stage := range stages {
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
// UTC で動きます（これが本テストの発端となった状態そのものです）。tzdata の導入は「このステージは
// ローカルタイムを持つ」という意思表示なので、その対で ENV TZ を要求します。
func TestDockerfileTzdataStagesDeclareTimeZone(t *testing.T) {
	t.Parallel()

	root := moduleRoot(t)
	stages := collectDockerfileStages(t, root, serverDockerfile)

	withTzdata := 0
	for _, stage := range stages {
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

// TestPostgresWorkflowsDeclareTimeZone は、PostgreSQL を起動する全ワークフローが TZ と PGTZ を
// 宣言していることを機械検証します。
//
// GitHub Actions はサービス定義をワークフロー間で共有できないため、同じ宣言が全ファイルに写経
// されます。写経の漏れは変数名での grep には映らず（宣言済みのファイルしか引っ掛からない）、
// 漏れたワークフローだけが UTC のクラスタ既定で走ります。判定を image 名側から行うことで、
// 「PostgreSQL を使うのに TZ を宣言していない」ワークフローを取りこぼしません。
func TestPostgresWorkflowsDeclareTimeZone(t *testing.T) {
	t.Parallel()

	root := moduleRoot(t)

	provisioning := 0
	for _, file := range globRepoFiles(t, root, workflowFilePatterns...) {
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
		"PostgreSQL を起動するワークフローを 1 件も抽出できず、検証が空振りする")
}

// collectDockerfileStages は、Dockerfile を FROM 単位のステージへ分け、ローカルタイムに関わる
// 宣言を取り出して返します。depguard が go/ast を禁じるためテキスト走査で行います（既存 architest と同方針）。
func collectDockerfileStages(t *testing.T, root, file string) []dockerfileStage {
	t.Helper()

	var stages []dockerfileStage
	for line := range strings.SplitSeq(readRepoFile(t, root, file), "\n") {
		if from := dockerfileFromRe.FindStringSubmatch(line); from != nil {
			name := from[1]
			if name == "" {
				name = "(名前なし)"
			}
			stages = append(stages, dockerfileStage{name: name})
			continue
		}
		if len(stages) == 0 {
			continue
		}

		current := &stages[len(stages)-1]
		if tz := dockerfileEnvTZRe.FindStringSubmatch(line); tz != nil {
			current.timeZone = tz[1]
			current.hasTimeZone = true
		}
		if dockerfileTzdataRe.MatchString(line) {
			current.installsTzdata = true
		}
	}

	require.NotEmptyf(t, stages, "%s からステージを 1 件も抽出できず、検証が空振りする", file)
	return stages
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
