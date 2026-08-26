package main

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeMigrations は、指定した名前の空マイグレーションファイルを持つ一時ディレクトリを作ります。
func writeMigrations(t *testing.T, names ...string) string {
	t.Helper()

	dir := t.TempDir()

	for _, name := range names {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(""), 0o600))
	}

	return dir
}

func Test_collectVersions(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("指定種別のファイルだけから連番を昇順で集める", func(t *testing.T) {
			t.Parallel()
			dir := writeMigrations(t,
				"000002_add_products.up.sql",
				"000001_create_users.up.sql",
				"000003_add_orders.down.sql",
			)

			got, err := collectVersions(dir, "up")
			require.NoError(t, err)
			assert.Equal(t, []string{"000001", "000002"}, got)
		})

		t.Run("マイグレーションが 1 件も無ければ空を返す", func(t *testing.T) {
			t.Parallel()

			got, err := collectVersions(t.TempDir(), "up")
			require.NoError(t, err)
			assert.Empty(t, got)
		})

		t.Run("名前にアンダースコアが複数あっても最初の区切りまでを連番とする", func(t *testing.T) {
			t.Parallel()
			dir := writeMigrations(t, "000001_create_users_table.up.sql")

			got, err := collectVersions(dir, "up")
			require.NoError(t, err)
			assert.Equal(t, []string{"000001"}, got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("連番の区切りが無いファイル名はエラーにする", func(t *testing.T) {
			t.Parallel()
			dir := writeMigrations(t, "000001.up.sql")

			_, err := collectVersions(dir, "up")
			require.ErrorIs(t, err, errNoVersionPrefix)
			assert.Contains(t, err.Error(), "no version prefix")
		})

		t.Run("走査パターンとして壊れたディレクトリ名は 0 件に寄せずエラーにする", func(t *testing.T) {
			t.Parallel()

			versions, err := collectVersions("[", "up")
			require.ErrorIs(t, err, filepath.ErrBadPattern)
			assert.Contains(t, err.Error(), "failed to scan [")
			assert.Nil(t, versions)
		})
	})
}

func Test_reportDuplicates(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("重複が無ければ報告しない", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, reportDuplicates("up", []string{"000001", "000002", "000003"}))
		})

		t.Run("空でも報告しない", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, reportDuplicates("up", nil))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("重複した連番を種別付きで報告する", func(t *testing.T) {
			t.Parallel()
			got := reportDuplicates("down", []string{"000001", "000002", "000002"})
			assert.Equal(t, "Duplicate migration numbers (down): 000002", got)
		})

		t.Run("3 個以上の重複でも連番は 1 度だけ挙げる", func(t *testing.T) {
			t.Parallel()
			got := reportDuplicates("up", []string{"000001", "000001", "000001"})
			assert.Equal(t, "Duplicate migration numbers (up): 000001", got)
		})

		t.Run("重複が複数種あればすべて挙げる", func(t *testing.T) {
			t.Parallel()
			got := reportDuplicates("up", []string{"000001", "000001", "000002", "000002"})
			assert.Equal(t, "Duplicate migration numbers (up): 000001 000002", got)
		})
	})
}

func Test_reportGaps(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("欠番が無ければ報告しない", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, reportGaps("up", []string{"000001", "000002", "000003"}))
		})

		t.Run("1 から始まらなくても連続していれば報告しない", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, reportGaps("up", []string{"000005", "000006"}))
		})

		t.Run("空でも報告しない（サンプル削除でマイグレーションが消え得るため）", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, reportGaps("up", nil))
		})

		t.Run("1 件だけなら報告しない", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, reportGaps("up", []string{"000007"}))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("欠番を検出して期待値と実在を並べて報告する", func(t *testing.T) {
			t.Parallel()
			got := reportGaps("up", []string{"000001", "000003"})
			assert.Contains(t, got, "Migration version gap detected (up)")
			assert.Contains(t, got, "Expected :\n000001\n000002\n000003")
			assert.Contains(t, got, "Existing :\n000001\n000003")
		})

		t.Run("連番が数値でなければ報告する", func(t *testing.T) {
			t.Parallel()
			got := reportGaps("up", []string{"00000a", "00000b"})
			assert.Contains(t, got, "not numeric")
		})
	})
}

func Test_expectedSequence(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("最小値の桁数へゼロ埋めして最大値まで並べる", func(t *testing.T) {
			t.Parallel()
			got, err := expectedSequence([]string{"000008", "000010"})
			require.NoError(t, err)
			assert.Equal(t, []string{"000008", "000009", "000010"}, got)
		})

		t.Run("ゼロ埋めの無い連番でもそのままの桁数を保つ", func(t *testing.T) {
			t.Parallel()
			got, err := expectedSequence([]string{"8", "10"})
			require.NoError(t, err)
			assert.Equal(t, []string{"8", "9", "10"}, got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("先頭が数値でなければ欠番を捏造せずエラーを返す", func(t *testing.T) {
			t.Parallel()
			_, err := expectedSequence([]string{"00000a", "000010"})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "00000a")
		})

		t.Run("末尾が数値でなければ欠番を捏造せずエラーを返す", func(t *testing.T) {
			t.Parallel()
			_, err := expectedSequence([]string{"000008", "00000b"})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "00000b")
		})
	})
}

// captureLog は、標準ロガーの出力先をバッファへ差し替え、その内容を読む関数を返します。
// 出力先はプロセス共通のため、使うテストは並列化できません。
func captureLog(t *testing.T) func() string {
	t.Helper()

	var buf bytes.Buffer

	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	return buf.String
}

func Test_run(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("重複も欠番も無ければ成功する", func(t *testing.T) {
			t.Parallel()
			dir := writeMigrations(t, "000001_a.up.sql", "000002_b.up.sql")

			require.NoError(t, run([]string{"-dir", dir, "-check", "duplicate"}))
			require.NoError(t, run([]string{"-dir", dir, "-check", "gap"}))
		})

		t.Run("種別の指定で検査対象のファイルを切り替える", func(t *testing.T) {
			t.Parallel()
			dir := writeMigrations(t, "000001_a.up.sql", "000001_b.up.sql", "000001_a.down.sql")

			require.NoError(t, run([]string{"-dir", dir, "-kind", "down"}))
		})

		t.Run("マイグレーションが 1 件も無ければ成功する", func(t *testing.T) {
			t.Parallel()

			require.NoError(t, run([]string{"-dir", t.TempDir(), "-check", "gap"}))
		})

		t.Run("ヘルプ要求は失敗にしない", func(t *testing.T) {
			t.Parallel()

			require.NoError(t, run([]string{"-h"}))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("既定では up の重複を検査する", func(t *testing.T) {
			t.Parallel()
			dir := writeMigrations(t, "000001_a.up.sql", "000001_b.up.sql")

			require.ErrorIs(t, run([]string{"-dir", dir}), errNumberingProblem)
		})

		t.Run("欠番を検出して失敗する", func(t *testing.T) {
			t.Parallel()
			dir := writeMigrations(t, "000001_a.up.sql", "000003_c.up.sql")

			require.ErrorIs(t, run([]string{"-dir", dir, "-check", "gap"}), errNumberingProblem)
		})

		//nolint:paralleltest // 標準ロガーの出力先を差し替えるため並列化できない
		t.Run("何がぶつかっているかを出力してから失敗する", func(t *testing.T) {
			dir := writeMigrations(t, "000001_a.up.sql", "000001_b.up.sql")
			logged := captureLog(t)

			require.ErrorIs(t, run([]string{"-dir", dir}), errNumberingProblem)
			assert.Contains(t, logged(), "Duplicate migration numbers (up): 000001")
		})

		t.Run("未知の検査内容は既定の検査へ流さず失敗する", func(t *testing.T) {
			t.Parallel()
			dir := writeMigrations(t, "000001_a.up.sql", "000001_b.up.sql")

			err := run([]string{"-dir", dir, "-check", "bogus"})

			require.ErrorIs(t, err, errUnknownCheck)
			assert.NotErrorIs(t, err, errNumberingProblem)
		})

		t.Run("連番を集められなければ 0 件に寄せず失敗する", func(t *testing.T) {
			t.Parallel()

			require.ErrorIs(t, run([]string{"-dir", "["}), filepath.ErrBadPattern)
		})

		t.Run("未知のフラグはヘルプ要求と混同せず失敗する", func(t *testing.T) {
			t.Parallel()

			require.ErrorContains(t, run([]string{"-bogus"}), "failed to parse flags")
		})
	})
}
