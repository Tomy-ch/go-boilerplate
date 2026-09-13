package architest

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// timeFormatCall は、event パッケージが時刻を自分で整形している呼び出しです。綴り方は
// internal/usecase/tools/datetime が 1 箇所で持つので、event パッケージにこの呼び出しは現れません。
const timeFormatCall = ".Format("

// TestOutboxPayloadTimeUTC は、outbox payload の時刻項目が datetime.FormatUTC を通ることを固定します。
// 契約とその根拠は internal/usecase/README.md の event/payload_parity.yaml 節を参照。
//
// 対象 0 件は許容します（sample API 撤去後は event パッケージごと消えるため、ここに canary は
// 置けません。internal/architest/README.md の Notes を参照）。
func TestOutboxPayloadTimeUTC(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("どの payload も時刻を自分で整形していない", func(t *testing.T) {
			t.Parallel()

			violations, err := collectOutboxPayloadTimeViolations(moduleRoot(t))
			require.NoError(t, err)
			assert.Empty(t, violations,
				"payload が時刻を自分で整形している。"+
					"internal/usecase/tools/datetime の FormatUTC を通すこと")
		})

		t.Run("event パッケージが 1 つも無いツリーは違反なしで通す", func(t *testing.T) {
			t.Parallel()

			violations, err := collectOutboxPayloadTimeViolations(t.TempDir())

			require.NoError(t, err)
			assert.Empty(t, violations)
		})

		t.Run("自分で整形している payload を検出する", func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			writeEventPackage(t, root, "alpha",
				"package event\n\nfunc BuildOpened(at time.Time) ([]byte, error) {\n"+
					"\ts := at.UTC().Format(time.RFC3339Nano)\n\treturn []byte(s), nil\n}\n")

			violations, err := collectOutboxPayloadTimeViolations(root)

			require.NoError(t, err)
			assert.Equal(t, []string{
				"internal/usecase/alpha/event/alpha_event.go:4: s := at.UTC().Format(time.RFC3339Nano)",
			}, violations)
		})

		t.Run("datetime を通している payload は検出しない", func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			writeEventPackage(t, root, "beta",
				"package event\n\nfunc BuildOpened(at time.Time) ([]byte, error) {\n"+
					"\ts := datetime.FormatUTC(at)\n\treturn []byte(s), nil\n}\n")

			violations, err := collectOutboxPayloadTimeViolations(root)

			require.NoError(t, err)
			assert.Empty(t, violations)
		})
	})
}

// collectOutboxPayloadTimeViolations は、root 配下の event パッケージを走査して、時刻を自分で整形して
// いる行を集めます。並びはパッケージ・ファイル・行番号の順で、同じツリーからは常に同じ結果になります。
func collectOutboxPayloadTimeViolations(root string) ([]string, error) {
	dirs, err := collectEventPackages(root)
	if err != nil {
		return nil, err
	}

	violations := []string{}
	for _, dir := range dirs {
		paths, perr := plainGoFiles(dir)
		if perr != nil {
			return nil, perr
		}

		for _, path := range paths {
			found, ferr := collectFileTimeViolations(root, path)
			if ferr != nil {
				return nil, ferr
			}
			violations = append(violations, found...)
		}
	}

	return violations, nil
}

// collectFileTimeViolations は、ファイル 1 つ分の違反行を `<相対パス>:<行番号>: <行>` の形で返します。
func collectFileTimeViolations(root, path string) ([]string, error) {
	lines, err := readLines(path)
	if err != nil {
		return nil, err
	}

	return timeViolationsInLines(relPath(root, path), lines), nil
}

// timeViolationsInLines は、時刻を自分で整形している行を `<rel>:<行番号>: <行>` の形で返します。
func timeViolationsInLines(rel string, lines []string) []string {
	violations := []string{}
	for i, line := range lines {
		if !strings.Contains(line, timeFormatCall) {
			continue
		}
		violations = append(violations, fmt.Sprintf("%s:%d: %s", rel, i+1, strings.TrimSpace(line)))
	}

	return violations
}

// Test_timeViolationsInLines は、走査ロジックを実ツリーではなく合成ソースで固定します
// （internal/architest/README.md の「Pin the scanning logic against synthetic sources」）。
func Test_timeViolationsInLines(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("自分で整形している行を行番号付きで返す", func(t *testing.T) {
			t.Parallel()

			got := timeViolationsInLines("a/b.go", []string{
				"package event",
				"",
				"\ts := at.UTC().Format(time.RFC3339Nano)",
			})

			assert.Equal(t, []string{"a/b.go:3: s := at.UTC().Format(time.RFC3339Nano)"}, got)
		})

		t.Run("レイアウトを変えても検出する", func(t *testing.T) {
			t.Parallel()

			got := timeViolationsInLines("a/b.go", []string{"\ts := at.Format(time.RFC3339)"})

			assert.Equal(t, []string{"a/b.go:1: s := at.Format(time.RFC3339)"}, got)
		})

		t.Run("レイアウトを次の行に置いた呼び出しも検出する", func(t *testing.T) {
			t.Parallel()

			got := timeViolationsInLines("a/b.go", []string{
				"\ts := at.Format(",
				"\t\ttime.RFC3339Nano,",
				"\t)",
			})

			assert.Equal(t, []string{"a/b.go:1: s := at.Format("}, got)
		})

		t.Run("datetime を通している行は返さない", func(t *testing.T) {
			t.Parallel()

			got := timeViolationsInLines("a/b.go", []string{"\ts := datetime.FormatUTC(at)"})

			assert.Empty(t, got)
		})

		t.Run("違反が複数あれば現れた順にすべて返す", func(t *testing.T) {
			t.Parallel()

			got := timeViolationsInLines("a/b.go", []string{
				"\tx := at.Format(time.RFC3339Nano)",
				"\ty := datetime.FormatUTC(at)",
				"\tz := at.Format(time.RFC3339)",
			})

			assert.Equal(t, []string{
				"a/b.go:1: x := at.Format(time.RFC3339Nano)",
				"a/b.go:3: z := at.Format(time.RFC3339)",
			}, got)
		})
	})
}
