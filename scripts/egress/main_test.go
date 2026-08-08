package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSSOT = `# comment
[class.base]
hosts = ["b1:443", "b2:443"]

[class.mise]
hosts = [
  "m1:443",
  "m2:443",
]

[class.image]
includes = ["mise"]
hosts = ["i1:443"]

[job."a.yaml:one"]
classes = ["image"]
extra = ["x1:443"]

[job."b.yaml:two"]
classes = []

[job."b.yaml:audited"]
egress_policy = "audit"
`

// bYAML は testSSOT が宣言する b.yaml 側の 2 ジョブ（block と audit）を持つ workflow。
const bYAML = `name: T

jobs:
  two:
    steps:
      - with:
          egress-policy: block
          allowed-endpoints: >
            b1:443
            b2:443

  audited:
    steps:
      - with:
          egress-policy: audit
`

// errWD は、作業ディレクトリの取得失敗の伝播を検証するためのセンチネルです。
var errWD = xerrors.New("getwd failed")

// testWorkflow は allowed-endpoints ブロックを 1 つ持つ workflow の雛形を返す。
func testWorkflow(jobID string, hosts ...string) string {
	var b strings.Builder
	b.WriteString("name: T\n\njobs:\n  " + jobID + ":\n    steps:\n")
	b.WriteString("      - name: Harden the runner\n        uses: step-security/harden-runner@x\n")
	b.WriteString("        with:\n          egress-policy: block\n          allowed-endpoints: >\n")
	for _, h := range hosts {
		b.WriteString("            " + h + "\n")
	}
	b.WriteString("\n      - name: Checkout\n")
	return b.String()
}

// newTestRepo は SSOT と workflow を持つ一時リポジトリを作り、その root を返す。
func newTestRepo(t *testing.T, workflows map[string]string) string {
	t.Helper()
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".github", "workflows"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(root, ssotFile), []byte(testSSOT), filePerm))
	for name, body := range workflows {
		require.NoError(t, os.WriteFile(filepath.Join(root, ".github", "workflows", name), []byte(body), filePerm))
	}
	return root
}

func mustParse(t *testing.T, body string) *ssot {
	t.Helper()
	s, err := parseSSOT(body)
	require.NoError(t, err)
	return s
}

func Test_run(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("apply はブロックを SSOT の内容へ書き換える", func(t *testing.T) {
			t.Parallel()
			root := newTestRepo(t, map[string]string{
				"a.yaml": testWorkflow("one", "stale:443"),
				"b.yaml": bYAML,
			})
			require.NoError(t, run([]string{"apply"}, func() (string, error) { return root, nil }))
			got, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "a.yaml")) //nolint:gosec // path from t.TempDir
			require.NoError(t, err)
			assert.Contains(t, string(got), "            b1:443\n")
			assert.NotContains(t, string(got), "stale:443")
		})

		t.Run("check はドリフトが無ければ成功する", func(t *testing.T) {
			t.Parallel()
			root := newTestRepo(t, map[string]string{
				"a.yaml": testWorkflow("one", "b1:443", "b2:443", "m1:443", "m2:443", "i1:443", "x1:443"),
				"b.yaml": bYAML,
			})
			assert.NoError(t, run([]string{"check"}, func() (string, error) { return root, nil }))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("サブコマンドが無ければ usage を返す", func(t *testing.T) {
			t.Parallel()
			err := run(nil, func() (string, error) { return "", nil })
			require.ErrorIs(t, err, errUsage)
		})

		t.Run("未知のサブコマンドは usage を返す", func(t *testing.T) {
			t.Parallel()
			err := run([]string{"resolve"}, func() (string, error) { return "", nil })
			require.ErrorIs(t, err, errUsage)
		})

		t.Run("作業ディレクトリの取得失敗を伝播する", func(t *testing.T) {
			t.Parallel()
			err := run([]string{"check"}, func() (string, error) { return "", errWD })
			require.ErrorIs(t, err, errWD)
		})
	})
}

func Test_parseSSOT(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("クラスとジョブを宣言順で読む", func(t *testing.T) {
			t.Parallel()
			s := mustParse(t, testSSOT)
			assert.Len(t, s.classes, 3)
			assert.Equal(t, []string{"a.yaml:one", "b.yaml:two", "b.yaml:audited"}, s.jobOrder)
			assert.Equal(t, []string{"b1:443", "b2:443"}, s.baseHosts)
			assert.Equal(t, []string{"mise"}, s.classes["image"].includes)
			assert.Equal(t, policyAudit, s.jobs["b.yaml:audited"].policy)
		})

		t.Run("egress_policy 未指定は block とみなす", func(t *testing.T) {
			t.Parallel()
			s := mustParse(t, testSSOT)
			assert.Equal(t, policyBlock, s.jobs["a.yaml:one"].policy)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("セクションの外の値を弾く", func(t *testing.T) {
			t.Parallel()
			_, err := parseSSOT(`hosts = ["x:443"]`)
			require.ErrorIs(t, err, errSSOTSyntax)
		})

		t.Run("解釈できない行を弾く", func(t *testing.T) {
			t.Parallel()
			_, err := parseSSOT("[class.base]\nhosts\n")
			require.ErrorIs(t, err, errSSOTSyntax)
		})

		t.Run("同一セクション内のキー重複を弾く", func(t *testing.T) {
			t.Parallel()
			_, err := parseSSOT("[class.base]\nhosts = [\"a:443\"]\nhosts = [\"b:443\"]\n")
			require.ErrorIs(t, err, errSSOTDuplicate)
		})
	})
}

func Test_stripComment(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("行末コメントを落とす", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, `hosts = ["a:443"] `, stripComment(`hosts = ["a:443"] # note`))
		})

		t.Run("コメントが無い行はそのまま返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "[class.base]", stripComment("[class.base]"))
		})
	})
}

func Test_parseAssignment(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("スカラーを読む", func(t *testing.T) {
			t.Parallel()
			key, values, next, err := parseAssignment([]string{`egress_policy = "audit"`}, 0)
			require.NoError(t, err)
			assert.Equal(t, "egress_policy", key)
			assert.Equal(t, []string{"audit"}, values)
			assert.Equal(t, 0, next)
		})

		t.Run("1 行の配列を読む", func(t *testing.T) {
			t.Parallel()
			key, values, next, err := parseAssignment([]string{`hosts = ["a:443", "b:80"]`}, 0)
			require.NoError(t, err)
			assert.Equal(t, "hosts", key)
			assert.Equal(t, []string{"a:443", "b:80"}, values)
			assert.Equal(t, 0, next)
		})

		t.Run("複数行の配列は閉じ括弧まで取り込む", func(t *testing.T) {
			t.Parallel()
			lines := []string{"hosts = [", `  "a:443",`, `  "b:80", # note`, "]"}
			key, values, next, err := parseAssignment(lines, 0)
			require.NoError(t, err)
			assert.Equal(t, "hosts", key)
			assert.Equal(t, []string{"a:443", "b:80"}, values)
			assert.Equal(t, 3, next)
		})

		t.Run("空配列を読む", func(t *testing.T) {
			t.Parallel()
			key, values, _, err := parseAssignment([]string{"classes = []"}, 0)
			require.NoError(t, err)
			assert.Equal(t, "classes", key)
			assert.Empty(t, values)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("代入として読めない行を弾く", func(t *testing.T) {
			t.Parallel()
			_, _, _, err := parseAssignment([]string{"hosts:"}, 0)
			require.ErrorIs(t, err, errSSOTSyntax)
		})

		t.Run("閉じられていない配列を弾く", func(t *testing.T) {
			t.Parallel()
			_, _, _, err := parseAssignment([]string{"hosts = [", `  "a:443",`}, 0)
			require.ErrorIs(t, err, errSSOTSyntax)
		})
	})
}

func Test_assign(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("クラスのキーを載せる", func(t *testing.T) {
			t.Parallel()
			c := &class{}
			require.NoError(t, assign(c, nil, "includes", []string{"mise"}))
			require.NoError(t, assign(c, nil, "hosts", []string{"a:443"}))
			assert.Equal(t, []string{"mise"}, c.includes)
			assert.Equal(t, []string{"a:443"}, c.hosts)
		})

		t.Run("ジョブのキーを載せる", func(t *testing.T) {
			t.Parallel()
			j := &jobSpec{}
			require.NoError(t, assign(nil, j, "classes", []string{"image"}))
			require.NoError(t, assign(nil, j, "extra", []string{"a:443"}))
			require.NoError(t, assign(nil, j, "egress_policy", []string{policyAudit}))
			assert.Equal(t, []string{"image"}, j.classes)
			assert.Equal(t, []string{"a:443"}, j.extra)
			assert.Equal(t, policyAudit, j.policy)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未知のキーを弾く", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, assign(&class{}, nil, "endpoints", []string{"a:443"}), errSSOTSyntax)
		})

		t.Run("クラスのキーをジョブへ書けない", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, assign(nil, &jobSpec{}, "includes", []string{"mise"}), errSSOTSyntax)
		})

		t.Run("リスト内の重複を弾く", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, assign(&class{}, nil, "hosts", []string{"a:443", "a:443"}), errSSOTDuplicate)
		})
	})
}

func Test_requireUnique(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("重複が無ければ成功する", func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, requireUnique([]string{"a", "b"}))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("重複を弾く", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, requireUnique([]string{"a", "b", "a"}), errSSOTDuplicate)
		})
	})
}

func Test_ssot_openSection(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("クラスとジョブを登録する", func(t *testing.T) {
			t.Parallel()
			s := &ssot{classes: map[string]*class{}, jobs: map[string]*jobSpec{}}
			require.NoError(t, s.openSection("class", "base"))
			require.NoError(t, s.openSection("job", "a.yaml:one"))
			assert.NotNil(t, s.classes["base"])
			assert.Equal(t, []string{"a.yaml:one"}, s.jobOrder)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("クラスの重複を弾く", func(t *testing.T) {
			t.Parallel()
			s := &ssot{classes: map[string]*class{}, jobs: map[string]*jobSpec{}}
			require.NoError(t, s.openSection("class", "base"))
			require.ErrorIs(t, s.openSection("class", "base"), errSSOTDuplicate)
		})

		t.Run("ジョブの重複を弾く", func(t *testing.T) {
			t.Parallel()
			s := &ssot{classes: map[string]*class{}, jobs: map[string]*jobSpec{}}
			require.NoError(t, s.openSection("job", "a.yaml:one"))
			require.ErrorIs(t, s.openSection("job", "a.yaml:one"), errSSOTDuplicate)
		})
	})
}

func Test_ssot_validate(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("base クラスの hosts を暗黙適用分として取り出す", func(t *testing.T) {
			t.Parallel()
			s := mustParse(t, testSSOT)
			assert.Equal(t, []string{"b1:443", "b2:443"}, s.baseHosts)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未定義クラスの includes を弾く", func(t *testing.T) {
			t.Parallel()
			_, err := parseSSOT("[class.image]\nincludes = [\"nope\"]\n")
			require.ErrorIs(t, err, errSSOTUnknownClass)
		})

		t.Run("クラスの循環継承を弾く", func(t *testing.T) {
			t.Parallel()
			body := "[class.a]\nincludes = [\"b\"]\n\n[class.b]\nincludes = [\"a\"]\n\n" +
				"[job.\"w.yaml:j\"]\nclasses = [\"a\"]\n"
			_, err := parseSSOT(body)
			require.ErrorIs(t, err, errSSOTClassCycle)
		})
	})
}

func Test_ssot_validateJob(t *testing.T) {
	t.Parallel()

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未知の egress_policy を弾く", func(t *testing.T) {
			t.Parallel()
			_, err := parseSSOT("[job.\"w.yaml:j\"]\negress_policy = \"warn\"\n")
			require.ErrorIs(t, err, errSSOTPolicy)
		})

		t.Run("audit ジョブが許可リストを持つことを弾く", func(t *testing.T) {
			t.Parallel()
			body := "[job.\"w.yaml:j\"]\negress_policy = \"audit\"\nextra = [\"a:443\"]\n"
			_, err := parseSSOT(body)
			require.ErrorIs(t, err, errSSOTPolicy)
		})

		t.Run("base の明示宣言を弾く", func(t *testing.T) {
			t.Parallel()
			body := "[class.base]\nhosts = [\"b:443\"]\n\n[job.\"w.yaml:j\"]\nclasses = [\"base\"]\n"
			_, err := parseSSOT(body)
			require.ErrorIs(t, err, errSSOTBaseClass)
		})

		t.Run("未定義クラスの宣言を弾く", func(t *testing.T) {
			t.Parallel()
			_, err := parseSSOT("[job.\"w.yaml:j\"]\nclasses = [\"nope\"]\n")
			require.ErrorIs(t, err, errSSOTUnknownClass)
		})
	})
}

func Test_ssot_hostsFor(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("base → クラス（継承込み）→ extra の順で並べる", func(t *testing.T) {
			t.Parallel()
			hosts, err := mustParse(t, testSSOT).hostsFor("a.yaml:one")
			require.NoError(t, err)
			assert.Equal(t, []string{"b1:443", "b2:443", "m1:443", "m2:443", "i1:443", "x1:443"}, hosts)
		})

		t.Run("クラス宣言が無いジョブは base だけを持つ", func(t *testing.T) {
			t.Parallel()
			hosts, err := mustParse(t, testSSOT).hostsFor("b.yaml:two")
			require.NoError(t, err)
			assert.Equal(t, []string{"b1:443", "b2:443"}, hosts)
		})

		t.Run("audit ジョブは許可リストを持たない", func(t *testing.T) {
			t.Parallel()
			hosts, err := mustParse(t, testSSOT).hostsFor("b.yaml:audited")
			require.NoError(t, err)
			assert.Empty(t, hosts)
		})

		t.Run("クラス間で重複するホストは初出だけを残す", func(t *testing.T) {
			t.Parallel()
			body := "[class.base]\nhosts = [\"b:443\"]\n\n[class.x]\nhosts = [\"b:443\", \"x:443\"]\n\n" +
				"[job.\"w.yaml:j\"]\nclasses = [\"x\"]\nextra = [\"x:443\"]\n"
			hosts, err := mustParse(t, body).hostsFor("w.yaml:j")
			require.NoError(t, err)
			assert.Equal(t, []string{"b:443", "x:443"}, hosts)
		})
	})
}

func Test_ssot_expandClass(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("継承元を先に展開する", func(t *testing.T) {
			t.Parallel()
			out := &orderedHosts{seen: map[string]bool{}}
			require.NoError(t, mustParse(t, testSSOT).expandClass("image", out, map[string]bool{}))
			assert.Equal(t, []string{"m1:443", "m2:443", "i1:443"}, out.list)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未定義クラスを弾く", func(t *testing.T) {
			t.Parallel()
			out := &orderedHosts{seen: map[string]bool{}}
			err := mustParse(t, testSSOT).expandClass("nope", out, map[string]bool{})
			require.ErrorIs(t, err, errSSOTUnknownClass)
		})

		t.Run("循環を弾く", func(t *testing.T) {
			t.Parallel()
			s := &ssot{classes: map[string]*class{"a": {includes: []string{"a"}}}, jobs: map[string]*jobSpec{}}
			out := &orderedHosts{seen: map[string]bool{}}
			require.ErrorIs(t, s.expandClass("a", out, map[string]bool{}), errSSOTClassCycle)
		})
	})
}

func Test_orderedHosts_add(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("初出順を保ったまま重複を落とす", func(t *testing.T) {
			t.Parallel()
			o := &orderedHosts{seen: map[string]bool{}}
			o.add([]string{"a", "b"})
			o.add([]string{"b", "c"})
			assert.Equal(t, []string{"a", "b", "c"}, o.list)
		})
	})
}

func Test_scanWorkflow(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ジョブごとの policy とブロックを返す", func(t *testing.T) {
			t.Parallel()
			body := testWorkflow("one", "a:443", "b:443")
			policies, blocks, err := scanWorkflow("a.yaml", strings.Split(body, "\n"))
			require.NoError(t, err)
			assert.Equal(t, map[string]string{"a.yaml:one": policyBlock}, policies)
			require.Len(t, blocks, 1)
			assert.Equal(t, "a.yaml:one", blocks[0].jobKey)
			assert.Equal(t, []string{"a:443", "b:443"}, blocks[0].hosts)
			assert.Equal(t, 10, blocks[0].indent)
		})

		t.Run("複数ジョブを job id で識別する", func(t *testing.T) {
			t.Parallel()
			body := testWorkflow("one", "a:443") + "\n" +
				strings.TrimPrefix(testWorkflow("two", "b:443"), "name: T\n\njobs:\n")
			policies, blocks, err := scanWorkflow("a.yaml", strings.Split(body, "\n"))
			require.NoError(t, err)
			assert.Len(t, policies, 2)
			require.Len(t, blocks, 2)
			assert.Equal(t, "a.yaml:two", blocks[1].jobKey)
		})

		t.Run("harden-runner を持たないジョブは policy を持たない", func(t *testing.T) {
			t.Parallel()
			body := "name: T\n\njobs:\n  notify-failure:\n    uses: ./.github/workflows/notify.yaml\n"
			policies, blocks, err := scanWorkflow("a.yaml", strings.Split(body, "\n"))
			require.NoError(t, err)
			assert.Empty(t, policies)
			assert.Empty(t, blocks)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ジョブに属さない allowed-endpoints を弾く", func(t *testing.T) {
			t.Parallel()
			body := "name: T\n\nx:\n  with:\n    allowed-endpoints: >\n      a:443\n"
			_, _, err := scanWorkflow("a.yaml", strings.Split(body, "\n"))
			require.ErrorIs(t, err, errWorkflowJobless)
		})

		t.Run("ジョブに属さない egress-policy を弾く", func(t *testing.T) {
			t.Parallel()
			body := "name: T\n\nx:\n  with:\n    egress-policy: block\n"
			_, _, err := scanWorkflow("a.yaml", strings.Split(body, "\n"))
			require.ErrorIs(t, err, errWorkflowJobless)
		})
	})
}

func Test_workflowJobs(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("jobs 直下の job id を行ごとに割り当てる", func(t *testing.T) {
			t.Parallel()
			lines := []string{"name: T", "on:", "  push:", "jobs:", "  one:", "    steps: []", "  two:", "    steps: []"}
			assert.Equal(t, []string{"", "", "", "", "one", "one", "two", "two"}, workflowJobs(lines))
		})

		t.Run("jobs の外は空文字にする", func(t *testing.T) {
			t.Parallel()
			lines := []string{"jobs:", "  one:", "on:", "  push:"}
			assert.Equal(t, []string{"", "one", "", ""}, workflowJobs(lines))
		})

		t.Run("桁 0 のコメント行は所属を変えない", func(t *testing.T) {
			t.Parallel()
			lines := []string{"jobs:", "  one:", "# note", "    steps: []"}
			assert.Equal(t, []string{"", "one", "one", "one"}, workflowJobs(lines))
		})
	})
}

func Test_readBlock(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("字下げが戻るまでをホスト行として読む", func(t *testing.T) {
			t.Parallel()
			lines := []string{"          allowed-endpoints: >", "            a:443", "            b:80", "", "      - name: X"}
			b, err := readBlock("a.yaml", lines, 0, 10)
			require.NoError(t, err)
			assert.Equal(t, []string{"a:443", "b:80"}, b.hosts)
			assert.Equal(t, 3, b.endLine)
		})

		t.Run("ホスト行が無いブロックを空として読む", func(t *testing.T) {
			t.Parallel()
			b, err := readBlock("a.yaml", []string{"          allowed-endpoints: >", ""}, 0, 10)
			require.NoError(t, err)
			assert.Empty(t, b.hosts)
			assert.Equal(t, 1, b.endLine)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("コメント行を弾く", func(t *testing.T) {
			t.Parallel()
			lines := []string{"          allowed-endpoints: >", "            # note", "            a:443"}
			_, err := readBlock("a.yaml", lines, 0, 10)
			require.ErrorIs(t, err, errBlockComment)
		})

		t.Run("1 行に複数トークンある行を弾く", func(t *testing.T) {
			t.Parallel()
			lines := []string{"          allowed-endpoints: >", "            a:443 b:443"}
			_, err := readBlock("a.yaml", lines, 0, 10)
			require.ErrorIs(t, err, errBlockComment)
		})
	})
}

func Test_renderBlock(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キーより 2 段深い字下げで並べる", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, []string{"            a:443", "            b:80"}, renderBlock([]string{"a:443", "b:80"}, 10))
		})

		t.Run("ホストが無ければ空を返す", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, renderBlock(nil, 10))
		})
	})
}

func Test_ssot_rewriteFile(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ブロックを SSOT 由来の内容へ差し替える", func(t *testing.T) {
			t.Parallel()
			used := map[string]bool{}
			out, err := mustParse(t, testSSOT).rewriteFile("a.yaml", testWorkflow("one", "stale:443"), used)
			require.NoError(t, err)
			assert.Contains(t, out, "            b1:443\n            b2:443\n            m1:443\n")
			assert.NotContains(t, out, "stale:443")
			assert.True(t, used["a.yaml:one"])
		})

		t.Run("audit ジョブのブロック無しを許す", func(t *testing.T) {
			t.Parallel()
			body := "name: T\n\njobs:\n  audited:\n    steps:\n      - with:\n          egress-policy: audit\n"
			used := map[string]bool{}
			out, err := mustParse(t, testSSOT).rewriteFile("b.yaml", body, used)
			require.NoError(t, err)
			assert.Equal(t, body, out)
			assert.True(t, used["b.yaml:audited"])
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("SSOT に未登録のジョブを弾く", func(t *testing.T) {
			t.Parallel()
			_, err := mustParse(t, testSSOT).rewriteFile("z.yaml", testWorkflow("nope", "a:443"), map[string]bool{})
			require.ErrorIs(t, err, errJobMissing)
		})
	})
}

func Test_ssot_checkPolicy(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("block とブロック有りが揃う", func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, mustParse(t, testSSOT).checkPolicy("a.yaml:one", policyBlock, true))
		})

		t.Run("audit とブロック無しが揃う", func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, mustParse(t, testSSOT).checkPolicy("b.yaml:audited", policyAudit, false))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未登録のジョブを弾く", func(t *testing.T) {
			t.Parallel()
			err := mustParse(t, testSSOT).checkPolicy("z.yaml:none", policyBlock, true)
			require.ErrorIs(t, err, errJobMissing)
		})

		t.Run("policy の食い違いを弾く", func(t *testing.T) {
			t.Parallel()
			err := mustParse(t, testSSOT).checkPolicy("a.yaml:one", policyAudit, false)
			require.ErrorIs(t, err, errPolicyMismatch)
		})

		t.Run("block なのにブロックが無い状態を弾く", func(t *testing.T) {
			t.Parallel()
			err := mustParse(t, testSSOT).checkPolicy("a.yaml:one", policyBlock, false)
			require.ErrorIs(t, err, errPolicyMismatch)
		})

		t.Run("audit なのにブロックがある状態を弾く", func(t *testing.T) {
			t.Parallel()
			err := mustParse(t, testSSOT).checkPolicy("b.yaml:audited", policyAudit, true)
			require.ErrorIs(t, err, errPolicyMismatch)
		})
	})
}

func Test_ssot_planFiles(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("差分のあるファイルだけを返す", func(t *testing.T) {
			t.Parallel()
			root := newTestRepo(t, map[string]string{
				"a.yaml": testWorkflow("one", "stale:443"),
				"b.yaml": testWorkflow("two", "b1:443", "b2:443"),
			})
			files, err := filepath.Glob(filepath.Join(root, ".github", "workflows", "*.yaml"))
			require.NoError(t, err)
			changes, used, err := mustParse(t, testSSOT).planFiles(files)
			require.NoError(t, err)
			assert.Len(t, changes, 1)
			assert.True(t, used["a.yaml:one"])
			assert.True(t, used["b.yaml:two"])
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("読めないファイルを弾く", func(t *testing.T) {
			t.Parallel()
			_, _, err := mustParse(t, testSSOT).planFiles([]string{filepath.Join(t.TempDir(), "missing.yaml")})
			require.ErrorIs(t, err, os.ErrNotExist)
		})
	})
}

func Test_orphanJobs(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("参照されなかったジョブキーを返す", func(t *testing.T) {
			t.Parallel()
			s := mustParse(t, testSSOT)
			assert.Equal(t, []string{"b.yaml:audited", "b.yaml:two"}, orphanJobs(s, map[string]bool{"a.yaml:one": true}))
		})

		t.Run("全て参照済みなら空を返す", func(t *testing.T) {
			t.Parallel()
			s := mustParse(t, testSSOT)
			used := map[string]bool{"a.yaml:one": true, "b.yaml:two": true, "b.yaml:audited": true}
			assert.Empty(t, orphanJobs(s, used))
		})
	})
}

func Test_applyOrCheck(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("apply は workflow を書き換える", func(t *testing.T) {
			t.Parallel()
			root := newTestRepo(t, map[string]string{
				"a.yaml": testWorkflow("one", "stale:443"),
				"b.yaml": bYAML,
			})
			require.NoError(t, applyOrCheck(root, false))
			assert.NoError(t, applyOrCheck(root, true))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("SSOT が無ければ弾く", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, applyOrCheck(t.TempDir(), true), os.ErrNotExist)
		})

		t.Run("check はドリフトを非ゼロ終了で報告する", func(t *testing.T) {
			t.Parallel()
			root := newTestRepo(t, map[string]string{
				"a.yaml": testWorkflow("one", "stale:443"),
				"b.yaml": bYAML,
			})
			require.ErrorIs(t, applyOrCheck(root, true), errEgressDrift)
		})

		t.Run("workflow に現れないジョブ宣言を弾く", func(t *testing.T) {
			t.Parallel()
			root := newTestRepo(t, map[string]string{"a.yaml": testWorkflow("one", "stale:443")})
			require.ErrorIs(t, applyOrCheck(root, true), errJobOrphan)
		})
	})
}

func Test_writeChanges(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("差分が無ければ dry-run は成功する", func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, writeChanges(t.TempDir(), map[string]string{}, true))
		})

		t.Run("内容を書き出す", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			path := filepath.Join(root, "a.yaml")
			require.NoError(t, writeChanges(root, map[string]string{path: "body"}, false))
			got, err := os.ReadFile(path) //nolint:gosec // path from t.TempDir
			require.NoError(t, err)
			assert.Equal(t, "body", string(got))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("dry-run は差分をドリフトとして報告する", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			err := writeChanges(root, map[string]string{filepath.Join(root, "a.yaml"): "body"}, true)
			require.ErrorIs(t, err, errEgressDrift)
			assert.Contains(t, err.Error(), "a.yaml")
		})

		t.Run("書き込み失敗を伝播する", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			err := writeChanges(root, map[string]string{filepath.Join(root, "missing", "a.yaml"): "body"}, false)
			require.ErrorIs(t, err, os.ErrNotExist)
		})
	})
}

func Test_relTo(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("root からの相対パスを返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, filepath.Join(".github", "workflows", "a.yaml"),
				relTo("/repo", filepath.Join("/repo", ".github", "workflows", "a.yaml")))
		})

		t.Run("相対化できなければ元のパスを返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "a.yaml", relTo("/repo", "a.yaml"))
		})
	})
}
