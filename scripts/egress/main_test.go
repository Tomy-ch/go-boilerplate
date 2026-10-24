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
[ip."127.0.0.100"]
reason = "resolver"

[class.base]
hosts = ["b1", "b2"]

[class.mise]
hosts = [
  "m1",
  "m2",
]

[class.image]
includes = ["mise"]
hosts = ["i1"]

[job."a.yaml:one"]
classes = ["image"]
extra = ["x1"]
sudo = "needs docker"

[job."b.yaml:two"]
classes = []

[job."b.yaml:audited"]
egress_policy = "audit"
`

const testUses = "uses: bullfrogsec/bullfrog@0123456789abcdef0123456789abcdef01234567 # v0.11.1"

// bYAML は testSSOT が宣言する b.yaml 側の 2 ジョブ（block と audit）を、生成済みの形で持つ workflow。
const bYAML = `name: T

jobs:
  two:
    steps:
      - name: Harden the runner
        ` + testUses + `
        with:
          egress-policy: block
          dns-policy: allowed-domains-only
          enable-sudo: false
          allowed-ips: |
            127.0.0.100
          allowed-domains: |
            b1
            b2
      - name: Checkout

  audited:
    steps:
      - name: Harden the runner
        ` + testUses + `
        with:
          egress-policy: audit
          enable-sudo: false
          allowed-ips: |
            127.0.0.100
`

// errWD は、作業ディレクトリの取得失敗の伝播を検証するためのセンチネルです。
var errWD = xerrors.New("getwd failed")

// testWorkflow は通信制限ステップを先頭に 1 つ持つジョブの workflow の雛形を返す。
// inner は with の中に置く行で、字下げは雛形が付ける。
func testWorkflow(jobID string, inner ...string) string {
	var b strings.Builder
	b.WriteString("name: T\n\njobs:\n  " + jobID + ":\n    steps:\n")
	b.WriteString("      - name: Harden the runner\n        " + testUses + "\n        with:\n")
	for _, l := range inner {
		b.WriteString("          " + l + "\n")
	}
	b.WriteString("      - name: Checkout\n        uses: actions/checkout@x\n")
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

func newSSOT() *ssot {
	return &ssot{classes: map[string]*class{}, jobs: map[string]*jobSpec{}, reasons: map[string]string{}}
}

func Test_run(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("apply は with ブロックを SSOT の内容へ書き換える", func(t *testing.T) {
			t.Parallel()
			root := newTestRepo(t, map[string]string{
				"a.yaml": testWorkflow("one", "egress-policy: block", "allowed-endpoints: >", "  stale:443"),
				"b.yaml": bYAML,
			})
			require.NoError(t, run([]string{"apply"}, func() (string, error) { return root, nil }))
			got, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "a.yaml")) //nolint:gosec // path from t.TempDir
			require.NoError(t, err)
			assert.Contains(t, string(got), "          allowed-domains: |\n            b1\n")
			assert.NotContains(t, string(got), "stale:443")
		})

		t.Run("check は生成済みの workflow を受け入れる", func(t *testing.T) {
			t.Parallel()
			root := newTestRepo(t, map[string]string{
				"a.yaml": testWorkflow("one", "egress-policy: block"),
				"b.yaml": bYAML,
			})
			require.NoError(t, run([]string{"apply"}, func() (string, error) { return root, nil }))
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

		t.Run("引数が 2 つ以上なら usage を返す", func(t *testing.T) {
			t.Parallel()
			err := run([]string{"apply", "extra"}, func() (string, error) { return "", nil })
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

		t.Run("IP とクラスとジョブを宣言順で読む", func(t *testing.T) {
			t.Parallel()
			s := mustParse(t, testSSOT)
			assert.Equal(t, []string{"127.0.0.100"}, s.ips)
			assert.Equal(t, "resolver", s.reasons["127.0.0.100"])
			assert.Len(t, s.classes, 3)
			assert.Equal(t, []string{"a.yaml:one", "b.yaml:two", "b.yaml:audited"}, s.jobOrder)
			assert.Equal(t, []string{"b1", "b2"}, s.baseHosts)
			assert.Equal(t, []string{"mise"}, s.classes["image"].includes)
			assert.Equal(t, "needs docker", s.jobs["a.yaml:one"].sudo)
			assert.Equal(t, policyAudit, s.jobs["b.yaml:audited"].policy)
		})

		t.Run("egress_policy 未指定は block とみなす", func(t *testing.T) {
			t.Parallel()
			s := mustParse(t, testSSOT)
			assert.Equal(t, policyBlock, s.jobs["a.yaml:one"].policy)
		})

		t.Run("sudo 未指定のジョブは sudo を外す", func(t *testing.T) {
			t.Parallel()
			s := mustParse(t, testSSOT)
			assert.Empty(t, s.jobs["b.yaml:two"].sudo)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("セクションの外の値を弾く", func(t *testing.T) {
			t.Parallel()
			_, err := parseSSOT(`hosts = ["x"]`)
			require.ErrorIs(t, err, errSSOTSyntax)
		})

		t.Run("解釈できない行を弾く", func(t *testing.T) {
			t.Parallel()
			_, err := parseSSOT("[class.base]\nhosts\n")
			require.ErrorIs(t, err, errSSOTSyntax)
		})

		t.Run("同一セクション内のキー重複を弾く", func(t *testing.T) {
			t.Parallel()
			_, err := parseSSOT("[class.base]\nhosts = [\"a\"]\nhosts = [\"b\"]\n")
			require.ErrorIs(t, err, errSSOTDuplicate)
		})

		t.Run("値の中の # を弾く", func(t *testing.T) {
			t.Parallel()
			_, err := parseSSOT("[job.\"w.yaml:j\"]\nsudo = \"see #12\"\n")
			require.ErrorIs(t, err, errSSOTSyntax)
		})

		t.Run("配列の継続行の値の中の # を弾く", func(t *testing.T) {
			t.Parallel()
			_, err := parseSSOT("[class.base]\nhosts = [\n  \"a#b\",\n]\n")
			require.ErrorIs(t, err, errSSOTSyntax)
		})

		t.Run("IP セクションの未知のキーを弾く", func(t *testing.T) {
			t.Parallel()
			_, err := parseSSOT("[ip.\"127.0.0.1\"]\nhosts = [\"a\"]\n")
			require.ErrorIs(t, err, errSSOTSyntax)
		})
	})
}

func Test_ssot_enterSection(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ジョブのセクションへの代入先を返す", func(t *testing.T) {
			t.Parallel()
			s := newSSOT()
			sec, err := s.enterSection("job", "a.yaml:one")
			require.NoError(t, err)
			assert.Same(t, s.jobs["a.yaml:one"], sec.job)
			assert.Nil(t, sec.class)
			assert.Empty(t, sec.ip)
		})

		t.Run("IP のセクションは名前を代入先にする", func(t *testing.T) {
			t.Parallel()
			sec, err := newSSOT().enterSection("ip", "127.0.0.1")
			require.NoError(t, err)
			assert.Equal(t, "127.0.0.1", sec.ip)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("登録の失敗を伝播する", func(t *testing.T) {
			t.Parallel()
			_, err := newSSOT().enterSection("ip", "localhost")
			require.ErrorIs(t, err, errSSOTIP)
		})
	})
}

func Test_section_assign(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ジョブのキーをジョブへ載せる", func(t *testing.T) {
			t.Parallel()
			sec, err := newSSOT().enterSection("job", "a.yaml:one")
			require.NoError(t, err)
			require.NoError(t, sec.assign("sudo", []string{"needs docker"}))
			assert.Equal(t, "needs docker", sec.job.sudo)
		})

		t.Run("IP のセクションでは reason を載せる", func(t *testing.T) {
			t.Parallel()
			s := newSSOT()
			sec, err := s.enterSection("ip", "127.0.0.1")
			require.NoError(t, err)
			require.NoError(t, sec.assign("reason", []string{"loopback"}))
			assert.Equal(t, "loopback", s.reasons["127.0.0.1"])
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("同じセクションの中のキーの重複を弾く", func(t *testing.T) {
			t.Parallel()
			sec, err := newSSOT().enterSection("class", "base")
			require.NoError(t, err)
			require.NoError(t, sec.assign("hosts", []string{"a"}))
			require.ErrorIs(t, sec.assign("hosts", []string{"b"}), errSSOTDuplicate)
		})
	})
}

func Test_stripValueComment(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("クオートの外の行末コメントを落とす", func(t *testing.T) {
			t.Parallel()
			got, err := stripValueComment(`hosts = ["a"] # note`, 0)
			require.NoError(t, err)
			assert.Equal(t, `hosts = ["a"] `, got)
		})

		t.Run("行末コメントの中のクオートは値とみなさない", func(t *testing.T) {
			t.Parallel()
			got, err := stripValueComment(`  "a", # note "b#c"`, 0)
			require.NoError(t, err)
			assert.Equal(t, `  "a", `, got)
		})

		t.Run("コメントが無い行はそのまま返す", func(t *testing.T) {
			t.Parallel()
			got, err := stripValueComment(`[class.base]`, 0)
			require.NoError(t, err)
			assert.Equal(t, "[class.base]", got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("クオートの中の # を弾く", func(t *testing.T) {
			t.Parallel()
			_, err := stripValueComment(`sudo = "see #12"`, 4)
			require.ErrorIs(t, err, errSSOTSyntax)
			assert.Contains(t, err.Error(), "5 行目")
		})
	})
}

func Test_stripComment(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("行末コメントを落とす", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, `hosts = ["a"] `, stripComment(`hosts = ["a"] # note`))
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
			key, values, next, err := parseAssignment([]string{`hosts = ["a", "b"]`}, 0)
			require.NoError(t, err)
			assert.Equal(t, "hosts", key)
			assert.Equal(t, []string{"a", "b"}, values)
			assert.Equal(t, 0, next)
		})

		t.Run("複数行の配列は閉じ括弧まで取り込む", func(t *testing.T) {
			t.Parallel()
			lines := []string{"hosts = [", `  "a",`, `  "b", # note`, "]"}
			key, values, next, err := parseAssignment(lines, 0)
			require.NoError(t, err)
			assert.Equal(t, "hosts", key)
			assert.Equal(t, []string{"a", "b"}, values)
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
			_, _, _, err := parseAssignment([]string{"hosts = [", `  "a",`}, 0)
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
			require.NoError(t, assign(c, nil, "hosts", []string{"a"}))
			assert.Equal(t, []string{"mise"}, c.includes)
			assert.Equal(t, []string{"a"}, c.hosts)
		})

		t.Run("ジョブのキーを載せる", func(t *testing.T) {
			t.Parallel()
			j := &jobSpec{}
			require.NoError(t, assign(nil, j, "classes", []string{"image"}))
			require.NoError(t, assign(nil, j, "extra", []string{"a"}))
			require.NoError(t, assign(nil, j, "egress_policy", []string{policyAudit}))
			require.NoError(t, assign(nil, j, "sudo", []string{"needs docker"}))
			assert.Equal(t, []string{"image"}, j.classes)
			assert.Equal(t, []string{"a"}, j.extra)
			assert.Equal(t, policyAudit, j.policy)
			assert.Equal(t, "needs docker", j.sudo)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未知のキーを弾く", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, assign(&class{}, nil, "endpoints", []string{"a"}), errSSOTSyntax)
		})

		t.Run("egress_policy に配列を書けない", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, assign(nil, &jobSpec{}, "egress_policy", []string{policyAudit, policyBlock}), errSSOTSyntax)
		})

		t.Run("sudo に空の配列を書けない", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, assign(nil, &jobSpec{}, "sudo", nil), errSSOTSyntax)
		})

		t.Run("クラスのキーをジョブへ書けない", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, assign(nil, &jobSpec{}, "includes", []string{"mise"}), errSSOTSyntax)
		})

		t.Run("sudo をクラスへ書けない", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, assign(&class{}, nil, "sudo", []string{"x"}), errSSOTSyntax)
		})

		t.Run("リスト内の重複を弾く", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, assign(&class{}, nil, "hosts", []string{"a", "a"}), errSSOTDuplicate)
		})

		t.Run("根拠の空な sudo を弾く", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, assign(nil, &jobSpec{}, "sudo", []string{" "}), errSSOTReason)
		})
	})
}

func Test_ssot_assignIP(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("reason を載せる", func(t *testing.T) {
			t.Parallel()
			s := newSSOT()
			require.NoError(t, s.assignIP("127.0.0.1", "reason", []string{"loopback"}))
			assert.Equal(t, "loopback", s.reasons["127.0.0.1"])
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("reason 以外のキーを弾く", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, newSSOT().assignIP("127.0.0.1", "hosts", []string{"a"}), errSSOTSyntax)
		})

		t.Run("空の reason を弾く", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, newSSOT().assignIP("127.0.0.1", "reason", []string{""}), errSSOTReason)
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

func Test_validateHost(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ドメイン名を受け入れる", func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, validateHost("api.github.com"))
		})

		t.Run("先頭のワイルドカードを受け入れる", func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, validateHost("*.blob.core.windows.net"))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ポート付きを弾く", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, validateHost("github.com:443"), errSSOTHost)
		})

		t.Run("途中のワイルドカードを弾く", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, validateHost("api.*.com"), errSSOTHost)
		})

		t.Run("bullfrog が受け付けない _ 入りを弾く", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, validateHost("_srv._tcp.example.com"), errSSOTHost)
		})

		t.Run("空白入りを弾く", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, validateHost("a b"), errSSOTHost)
		})
	})
}

func Test_validateIP(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("IPv4 のアドレスを受け入れる", func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, validateIP("127.0.0.100"))
		})

		t.Run("CIDR を受け入れる", func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, validateIP("172.16.0.0/12"))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ドメイン名を弾く", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, validateIP("github.com"), errSSOTIP)
		})

		t.Run("bullfrog が受け付けない IPv6 を弾く", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, validateIP("::1"), errSSOTIP)
		})

		t.Run("数字とドットだけでも IP として読めない値を弾く", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, validateIP("1.2.3"), errSSOTIP)
		})
	})
}

func Test_ssot_openSection(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("クラスとジョブと IP を登録する", func(t *testing.T) {
			t.Parallel()
			s := newSSOT()
			require.NoError(t, s.openSection("class", "base"))
			require.NoError(t, s.openSection("job", "a.yaml:one"))
			require.NoError(t, s.openSection("ip", "127.0.0.1"))
			assert.NotNil(t, s.classes["base"])
			assert.Equal(t, []string{"a.yaml:one"}, s.jobOrder)
			assert.Equal(t, []string{"127.0.0.1"}, s.ips)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("クラスの重複を弾く", func(t *testing.T) {
			t.Parallel()
			s := newSSOT()
			require.NoError(t, s.openSection("class", "base"))
			require.ErrorIs(t, s.openSection("class", "base"), errSSOTDuplicate)
		})

		t.Run("ジョブの重複を弾く", func(t *testing.T) {
			t.Parallel()
			s := newSSOT()
			require.NoError(t, s.openSection("job", "a.yaml:one"))
			require.ErrorIs(t, s.openSection("job", "a.yaml:one"), errSSOTDuplicate)
		})

		t.Run("IP の重複を弾く", func(t *testing.T) {
			t.Parallel()
			s := newSSOT()
			require.NoError(t, s.openSection("ip", "127.0.0.1"))
			require.ErrorIs(t, s.openSection("ip", "127.0.0.1"), errSSOTDuplicate)
		})

		t.Run("IP でない名前の IP セクションを弾く", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, newSSOT().openSection("ip", "localhost"), errSSOTIP)
		})
	})
}

func Test_ssot_validate(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("base クラスが無ければ暗黙適用分を持たない", func(t *testing.T) {
			t.Parallel()
			s := mustParse(t, "[job.\"w.yaml:j\"]\nclasses = []\n")
			assert.Empty(t, s.baseHosts)
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

		t.Run("reason の無い IP を弾く", func(t *testing.T) {
			t.Parallel()
			_, err := parseSSOT("[ip.\"127.0.0.1\"]\n")
			require.ErrorIs(t, err, errSSOTReason)
		})

		t.Run("クラスの不正なホストを弾く", func(t *testing.T) {
			t.Parallel()
			_, err := parseSSOT("[class.base]\nhosts = [\"github.com:443\"]\n")
			require.ErrorIs(t, err, errSSOTHost)
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
			body := "[job.\"w.yaml:j\"]\negress_policy = \"audit\"\nextra = [\"a\"]\n"
			_, err := parseSSOT(body)
			require.ErrorIs(t, err, errSSOTPolicy)
		})

		t.Run("base の明示宣言を弾く", func(t *testing.T) {
			t.Parallel()
			body := "[class.base]\nhosts = [\"b\"]\n\n[job.\"w.yaml:j\"]\nclasses = [\"base\"]\n"
			_, err := parseSSOT(body)
			require.ErrorIs(t, err, errSSOTBaseClass)
		})

		t.Run("未定義クラスの宣言を弾く", func(t *testing.T) {
			t.Parallel()
			_, err := parseSSOT("[job.\"w.yaml:j\"]\nclasses = [\"nope\"]\n")
			require.ErrorIs(t, err, errSSOTUnknownClass)
		})

		t.Run("extra の不正なホストを弾く", func(t *testing.T) {
			t.Parallel()
			_, err := parseSSOT("[job.\"w.yaml:j\"]\nextra = [\"a.com:80\"]\n")
			require.ErrorIs(t, err, errSSOTHost)
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
			assert.Equal(t, []string{"b1", "b2", "m1", "m2", "i1", "x1"}, hosts)
		})

		t.Run("クラス宣言が無いジョブは base だけを持つ", func(t *testing.T) {
			t.Parallel()
			hosts, err := mustParse(t, testSSOT).hostsFor("b.yaml:two")
			require.NoError(t, err)
			assert.Equal(t, []string{"b1", "b2"}, hosts)
		})

		t.Run("audit ジョブは許可リストを持たない", func(t *testing.T) {
			t.Parallel()
			hosts, err := mustParse(t, testSSOT).hostsFor("b.yaml:audited")
			require.NoError(t, err)
			assert.Empty(t, hosts)
		})

		t.Run("クラス間で重複するホストは初出だけを残す", func(t *testing.T) {
			t.Parallel()
			body := "[class.base]\nhosts = [\"b\"]\n\n[class.x]\nhosts = [\"b\", \"x\"]\n\n" +
				"[job.\"w.yaml:j\"]\nclasses = [\"x\"]\nextra = [\"x\"]\n"
			hosts, err := mustParse(t, body).hostsFor("w.yaml:j")
			require.NoError(t, err)
			assert.Equal(t, []string{"b", "x"}, hosts)
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
			assert.Equal(t, []string{"m1", "m2", "i1"}, out.list)
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

		t.Run("ジョブごとのステップの位置と policy を返す", func(t *testing.T) {
			t.Parallel()
			body := testWorkflow("one", "egress-policy: block", "allowed-domains: |", "  a")
			steps, err := scanWorkflow("a.yaml", strings.Split(body, "\n"))
			require.NoError(t, err)
			require.Len(t, steps, 1)
			assert.Equal(t, "a.yaml:one", steps[0].jobKey)
			assert.Equal(t, policyBlock, steps[0].policy)
			assert.Equal(t, 8, steps[0].indent)
			assert.Equal(t, 7, steps[0].withLine)
			assert.Equal(t, 11, steps[0].endLine)
		})

		t.Run("複数ジョブを job id で識別する", func(t *testing.T) {
			t.Parallel()
			body := testWorkflow("one", "egress-policy: block") + "\n" +
				strings.TrimPrefix(testWorkflow("two", "egress-policy: audit"), "name: T\n\njobs:\n")
			steps, err := scanWorkflow("a.yaml", strings.Split(body, "\n"))
			require.NoError(t, err)
			require.Len(t, steps, 2)
			assert.Equal(t, "a.yaml:two", steps[1].jobKey)
			assert.Equal(t, policyAudit, steps[1].policy)
		})

		t.Run("ステップを持たないジョブには求めない", func(t *testing.T) {
			t.Parallel()
			body := "name: T\n\njobs:\n  notify-failure:\n    uses: ./.github/workflows/notify.yaml\n"
			steps, err := scanWorkflow("a.yaml", strings.Split(body, "\n"))
			require.NoError(t, err)
			assert.Empty(t, steps)
		})

		t.Run("コメントでの言及は形を問わない", func(t *testing.T) {
			t.Parallel()
			body := "# bullfrogsec/bullfrog は各ジョブの先頭\n" + strings.Replace(testWorkflow("one", "egress-policy: block"),
				"uses: actions/checkout@x\n", "uses: actions/checkout@x # bullfrogsec/bullfrog より後\n", 1)
			steps, err := scanWorkflow("a.yaml", strings.Split(body, "\n"))
			require.NoError(t, err)
			assert.Len(t, steps, 1)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ステップを持つのに通信制限ステップの無いジョブを弾く", func(t *testing.T) {
			t.Parallel()
			body := "name: T\n\njobs:\n  one:\n    steps:\n      - name: Checkout\n        uses: actions/checkout@x\n"
			_, err := scanWorkflow("a.yaml", strings.Split(body, "\n"))
			require.ErrorIs(t, err, errStepMissing)
		})

		t.Run("旧 action の再混入を弾く", func(t *testing.T) {
			t.Parallel()
			body := "name: T\n\njobs:\n  one:\n    steps:\n      - uses: step-security/harden-runner@x\n"
			_, err := scanWorkflow("a.yaml", strings.Split(body, "\n"))
			require.ErrorIs(t, err, errStepMalformed)
		})

		t.Run("SHA で固定していない bullfrog を弾く", func(t *testing.T) {
			t.Parallel()
			body := "name: T\n\njobs:\n  one:\n    steps:\n      - uses: bullfrogsec/bullfrog@v0.11.1\n"
			_, err := scanWorkflow("a.yaml", strings.Split(body, "\n"))
			require.ErrorIs(t, err, errStepMalformed)
		})

		t.Run("1 ジョブに 2 つある通信制限ステップを弾く", func(t *testing.T) {
			t.Parallel()
			body := strings.Replace(testWorkflow("one", "egress-policy: block"),
				"      - name: Checkout\n", "      - "+testUses+"\n        with:\n          egress-policy: block\n", 1)
			_, err := scanWorkflow("a.yaml", strings.Split(body, "\n"))
			require.ErrorIs(t, err, errStepDuplicate)
		})

		t.Run("先頭にない通信制限ステップを弾く", func(t *testing.T) {
			t.Parallel()
			body := "name: T\n\njobs:\n  one:\n    steps:\n      - name: Checkout\n        uses: actions/checkout@x\n" +
				"      - name: Harden the runner\n        " + testUses + "\n        with:\n          egress-policy: block\n"
			_, err := scanWorkflow("a.yaml", strings.Split(body, "\n"))
			require.ErrorIs(t, err, errStepNotFirst)
		})

		t.Run("ジョブに属さない通信制限ステップを弾く", func(t *testing.T) {
			t.Parallel()
			body := "name: T\n\nx:\n  " + testUses + "\n  with:\n    egress-policy: block\n"
			_, err := scanWorkflow("a.yaml", strings.Split(body, "\n"))
			require.ErrorIs(t, err, errWorkflowJobless)
		})

		t.Run("実行を条件で外せる通信制限ステップを弾く", func(t *testing.T) {
			t.Parallel()
			body := strings.Replace(testWorkflow("one", "egress-policy: block"),
				"        "+testUses+"\n", "        if: false\n        "+testUses+"\n", 1)
			_, err := scanWorkflow("a.yaml", strings.Split(body, "\n"))
			require.ErrorIs(t, err, errStepMalformed)
		})

		t.Run("with の読めないステップを伝播する", func(t *testing.T) {
			t.Parallel()
			body := testWorkflow("one", "allowed-ips: |")
			_, err := scanWorkflow("a.yaml", strings.Split(body, "\n"))
			require.ErrorIs(t, err, errStepMalformed)
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

		t.Run("行末コメントの付いた job id も読む", func(t *testing.T) {
			t.Parallel()
			lines := []string{"jobs:", "  one: # note", "    steps: []"}
			assert.Equal(t, []string{"", "one", "one"}, workflowJobs(lines))
		})

		t.Run("桁 0 のコメント行は所属を変えない", func(t *testing.T) {
			t.Parallel()
			lines := []string{"jobs:", "  one:", "# note", "    steps: []"}
			assert.Equal(t, []string{"", "one", "one", "one"}, workflowJobs(lines))
		})
	})
}

func Test_jobsWithSteps(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("steps を持つジョブと先頭のステップの行を返す", func(t *testing.T) {
			t.Parallel()
			lines := []string{"jobs:", "  one:", "    steps:", "      # note", "      - name: A", "  two:", "    uses: ./x.yaml"}
			assert.Equal(t, map[string]int{"one": 4}, jobsWithSteps(lines, workflowJobs(lines)))
		})
	})
}

func Test_withinFirstStep(t *testing.T) {
	t.Parallel()

	lines := []string{"    steps:", "      - name: A", "        uses: a", "      - name: B", "        uses: b"}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("先頭のステップの中なら真を返す", func(t *testing.T) {
			t.Parallel()
			assert.True(t, withinFirstStep(lines, 1, 2))
		})

		t.Run("先頭のステップの項目の行そのものも中とみなす", func(t *testing.T) {
			t.Parallel()
			assert.True(t, withinFirstStep(lines, 1, 1))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("2 つ目以降のステップなら偽を返す", func(t *testing.T) {
			t.Parallel()
			assert.False(t, withinFirstStep(lines, 1, 4))
		})

		t.Run("先頭のステップより前なら偽を返す", func(t *testing.T) {
			t.Parallel()
			assert.False(t, withinFirstStep(lines, 1, 0))
		})
	})
}

func Test_readStep(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("字下げが戻るまでを with ブロックとして読む", func(t *testing.T) {
			t.Parallel()
			lines := []string{
				"        with:",
				"          egress-policy: audit",
				"          allowed-ips: |",
				"            127.0.0.1",
				"      - name: X",
			}
			st, err := readStep("a.yaml", lines, 0, 8)
			require.NoError(t, err)
			assert.Equal(t, policyAudit, st.policy)
			assert.Equal(t, 0, st.withLine)
			assert.Equal(t, 4, st.endLine)
		})

		t.Run("ブロックの後ろの空行とコメントは次のステップとの区切りとして扱う", func(t *testing.T) {
			t.Parallel()
			lines := []string{"        with:", "          egress-policy: audit", "", "      # note", "      - name: X"}
			st, err := readStep("a.yaml", lines, 0, 8)
			require.NoError(t, err)
			assert.Equal(t, 2, st.endLine)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("途中の空行で切れて残りが書き換えから漏れる with を弾く", func(t *testing.T) {
			t.Parallel()
			lines := []string{"        with:", "          egress-policy: block", "", "          allowed-domains: |"}
			_, err := readStep("a.yaml", lines, 0, 8)
			require.ErrorIs(t, err, errStepMalformed)
		})

		t.Run("uses の次に with が無いステップを弾く", func(t *testing.T) {
			t.Parallel()
			_, err := readStep("a.yaml", []string{"        id: x"}, 0, 8)
			require.ErrorIs(t, err, errStepMalformed)
		})

		t.Run("ファイル末尾の uses を弾く", func(t *testing.T) {
			t.Parallel()
			_, err := readStep("a.yaml", nil, 0, 8)
			require.ErrorIs(t, err, errStepMalformed)
		})

		t.Run("egress-policy の無い with を弾く", func(t *testing.T) {
			t.Parallel()
			_, err := readStep("a.yaml", []string{"        with:", "          enable-sudo: false"}, 0, 8)
			require.ErrorIs(t, err, errStepMalformed)
		})
	})
}

func Test_blockResumes(t *testing.T) {
	t.Parallel()

	const inner = "          "

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("空行とコメントの後ろに同じ字下げの行が続けば真を返す", func(t *testing.T) {
			t.Parallel()
			assert.True(t, blockResumes([]string{"", "    # note", inner + "a: b"}, 0, inner))
		})

		t.Run("空行の後ろが浅い行なら偽を返す", func(t *testing.T) {
			t.Parallel()
			assert.False(t, blockResumes([]string{"", "      - name: X"}, 0, inner))
		})

		t.Run("区切りなしで浅い行が来たら偽を返す", func(t *testing.T) {
			t.Parallel()
			assert.False(t, blockResumes([]string{"      - name: X"}, 0, inner))
		})

		t.Run("ファイル末尾までなら偽を返す", func(t *testing.T) {
			t.Parallel()
			assert.False(t, blockResumes([]string{"", ""}, 0, inner))
		})
	})
}

func Test_checkStepKeys(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("name と id と uses と with だけのステップを受け入れる", func(t *testing.T) {
			t.Parallel()
			lines := []string{
				"      - name: H",
				"        id: h",
				"        " + testUses,
				"        with:",
				"          egress-policy: block",
				"      - name: Checkout",
			}
			assert.NoError(t, checkStepKeys(lines, 0, 5, 8))
		})

		t.Run("ステップの後ろの浅い行で検査を終える", func(t *testing.T) {
			t.Parallel()
			lines := []string{"      - " + testUses, "        with:", "          egress-policy: block", "  two:", "    if: x"}
			assert.NoError(t, checkStepKeys(lines, 0, 3, 8))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("uses の前の if を弾く", func(t *testing.T) {
			t.Parallel()
			lines := []string{"      - name: H", "        if: false", "        " + testUses, "        with:", "          egress-policy: block"}
			require.ErrorIs(t, checkStepKeys(lines, 0, 5, 8), errStepMalformed)
		})

		t.Run("with の後ろの continue-on-error を弾く", func(t *testing.T) {
			t.Parallel()
			lines := []string{"      - " + testUses, "        with:", "          egress-policy: block", "        continue-on-error: true"}
			require.ErrorIs(t, checkStepKeys(lines, 0, 3, 8), errStepMalformed)
		})
	})
}

func Test_renderStep(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("block は名前解決を絞り、IP とドメインを | の一覧で書く", func(t *testing.T) {
			t.Parallel()
			got := renderStep(&jobSpec{policy: policyBlock}, []string{"127.0.0.1"}, []string{"a", "*.b"}, 2)
			assert.Equal(t, []string{
				"  with:",
				"    egress-policy: block",
				"    dns-policy: allowed-domains-only",
				"    enable-sudo: false",
				"    allowed-ips: |",
				"      127.0.0.1",
				"    allowed-domains: |",
				"      a",
				"      *.b",
			}, got)
		})

		t.Run("sudo の根拠を持つジョブは sudo を残す", func(t *testing.T) {
			t.Parallel()
			got := renderStep(&jobSpec{policy: policyBlock, sudo: "needs docker"}, nil, []string{"a"}, 0)
			assert.Contains(t, got, "  enable-sudo: true")
		})

		t.Run("audit は名前解決とドメインを書かない", func(t *testing.T) {
			t.Parallel()
			got := renderStep(&jobSpec{policy: policyAudit}, []string{"127.0.0.1"}, nil, 0)
			assert.Equal(t, []string{"with:", "  egress-policy: audit", "  enable-sudo: false", "  allowed-ips: |", "    127.0.0.1"}, got)
		})

		t.Run("IP が無ければ allowed-ips を書かない", func(t *testing.T) {
			t.Parallel()
			got := renderStep(&jobSpec{policy: policyAudit}, nil, nil, 0)
			assert.Equal(t, []string{"with:", "  egress-policy: audit", "  enable-sudo: false"}, got)
		})
	})
}

func Test_ssot_rewriteFile(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("with ブロックを SSOT 由来の内容へ差し替える", func(t *testing.T) {
			t.Parallel()
			used := map[string]bool{}
			body := testWorkflow("one", "egress-policy: block", "allowed-endpoints: >", "  stale:443")
			out, err := mustParse(t, testSSOT).rewriteFile("a.yaml", body, used)
			require.NoError(t, err)
			assert.Contains(t, out, "          enable-sudo: true\n          allowed-ips: |\n            127.0.0.100\n")
			assert.Contains(t, out, "            i1\n            x1\n      - name: Checkout\n")
			assert.NotContains(t, out, "stale:443")
			assert.True(t, used["a.yaml:one"])
		})

		t.Run("行数の変わるステップが複数あっても互いの位置を崩さない", func(t *testing.T) {
			t.Parallel()
			body := strings.Replace(bYAML, `          egress-policy: block
          dns-policy: allowed-domains-only
          enable-sudo: false
          allowed-ips: |
            127.0.0.100
          allowed-domains: |
            b1
            b2
`, `          egress-policy: block
          allowed-endpoints: >
            stale1:443
            stale2:443
            stale3:443
            stale4:443
            stale5:443
            stale6:443
            stale7:443
`, 1)
			body = strings.Replace(body, `          egress-policy: audit
          enable-sudo: false
          allowed-ips: |
            127.0.0.100
`, `          egress-policy: audit
`, 1)
			require.NotEqual(t, bYAML, body)
			out, err := mustParse(t, testSSOT).rewriteFile("b.yaml", body, map[string]bool{})
			require.NoError(t, err)
			assert.Equal(t, strings.Split(bYAML, "\n"), strings.Split(out, "\n"))
		})

		t.Run("生成済みの workflow はそのまま返す", func(t *testing.T) {
			t.Parallel()
			used := map[string]bool{}
			out, err := mustParse(t, testSSOT).rewriteFile("b.yaml", bYAML, used)
			require.NoError(t, err)
			assert.Equal(t, strings.Split(bYAML, "\n"), strings.Split(out, "\n"))
			assert.True(t, used["b.yaml:two"])
			assert.True(t, used["b.yaml:audited"])
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("SSOT に未登録のジョブを弾く", func(t *testing.T) {
			t.Parallel()
			_, err := mustParse(t, testSSOT).rewriteFile("z.yaml", testWorkflow("nope", "egress-policy: block"), map[string]bool{})
			require.ErrorIs(t, err, errJobMissing)
			assert.Contains(t, err.Error(), "z.yaml:nope")
		})

		t.Run("走査の失敗を伝播する", func(t *testing.T) {
			t.Parallel()
			body := "name: T\n\njobs:\n  one:\n    steps:\n      - name: Checkout\n"
			_, err := mustParse(t, testSSOT).rewriteFile("a.yaml", body, map[string]bool{})
			require.ErrorIs(t, err, errStepMissing)
		})
	})
}

func Test_ssot_checkPolicy(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("block 同士が揃う", func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, mustParse(t, testSSOT).checkPolicy("a.yaml:one", policyBlock))
		})

		t.Run("audit 同士が揃う", func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, mustParse(t, testSSOT).checkPolicy("b.yaml:audited", policyAudit))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未登録のジョブを弾く", func(t *testing.T) {
			t.Parallel()
			err := mustParse(t, testSSOT).checkPolicy("z.yaml:none", policyBlock)
			require.ErrorIs(t, err, errJobMissing)
		})

		t.Run("policy の食い違いを弾く", func(t *testing.T) {
			t.Parallel()
			err := mustParse(t, testSSOT).checkPolicy("a.yaml:one", policyAudit)
			require.ErrorIs(t, err, errPolicyMismatch)
			assert.Contains(t, err.Error(), "a.yaml:one: workflow=audit SSOT=block")
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
				"a.yaml": testWorkflow("one", "egress-policy: block"),
				"b.yaml": bYAML,
			})
			files, err := filepath.Glob(filepath.Join(root, ".github", "workflows", "*.yaml"))
			require.NoError(t, err)
			changes, used, err := mustParse(t, testSSOT).planFiles(files)
			require.NoError(t, err)
			assert.Len(t, changes, 1)
			assert.Contains(t, changes, filepath.Join(root, ".github", "workflows", "a.yaml"))
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

		t.Run("apply は workflow を書き換え、その結果は check を通る", func(t *testing.T) {
			t.Parallel()
			root := newTestRepo(t, map[string]string{
				"a.yaml": testWorkflow("one", "egress-policy: block"),
				"b.yaml": bYAML,
			})
			require.NoError(t, applyOrCheck(root, false))
			assert.NoError(t, applyOrCheck(root, true))
		})

		t.Run("拡張子が yml の workflow も書き換える", func(t *testing.T) {
			t.Parallel()
			root := newTestRepo(t, map[string]string{
				"a.yml":  testWorkflow("one", "egress-policy: block"),
				"b.yaml": bYAML,
			})
			ssotBody := strings.Replace(testSSOT, `[job."a.yaml:one"]`, `[job."a.yml:one"]`, 1)
			require.NoError(t, os.WriteFile(filepath.Join(root, ssotFile), []byte(ssotBody), filePerm))
			require.NoError(t, applyOrCheck(root, false))
			got, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "a.yml")) //nolint:gosec // path from t.TempDir
			require.NoError(t, err)
			assert.Contains(t, string(got), "          enable-sudo: true\n")
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
				"a.yaml": testWorkflow("one", "egress-policy: block"),
				"b.yaml": bYAML,
			})
			require.ErrorIs(t, applyOrCheck(root, true), errEgressDrift)
		})

		t.Run("workflow に現れないジョブ宣言を弾く", func(t *testing.T) {
			t.Parallel()
			root := newTestRepo(t, map[string]string{"a.yaml": testWorkflow("one", "egress-policy: block")})
			err := applyOrCheck(root, true)
			require.ErrorIs(t, err, errJobOrphan)
			assert.Contains(t, err.Error(), "b.yaml:audited, b.yaml:two")
		})

		t.Run("孤児の宣言で失敗した apply は何も書かない", func(t *testing.T) {
			t.Parallel()
			before := testWorkflow("one", "egress-policy: block")
			root := newTestRepo(t, map[string]string{"a.yaml": before})
			require.ErrorIs(t, applyOrCheck(root, false), errJobOrphan)
			got, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "a.yaml")) //nolint:gosec // path from t.TempDir
			require.NoError(t, err)
			assert.Equal(t, before, string(got))
		})

		t.Run("別のファイルの走査で失敗した apply は書き換え予定のファイルにも触れない", func(t *testing.T) {
			t.Parallel()
			before := testWorkflow("one", "egress-policy: block")
			root := newTestRepo(t, map[string]string{
				"a.yaml": before,
				"b.yaml": bYAML,
				"c.yaml": "name: T\n\njobs:\n  three:\n    steps:\n      - name: Checkout\n",
			})
			require.ErrorIs(t, applyOrCheck(root, false), errStepMissing)
			got, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "a.yaml")) //nolint:gosec // path from t.TempDir
			require.NoError(t, err)
			assert.Equal(t, before, string(got))
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

		t.Run("dry-run は複数の差分をパスの昇順で並べる", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			changes := map[string]string{filepath.Join(root, "b.yaml"): "body", filepath.Join(root, "a.yaml"): "body"}
			err := writeChanges(root, changes, true)
			require.ErrorIs(t, err, errEgressDrift)
			assert.Contains(t, err.Error(), "a.yaml, b.yaml")
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
