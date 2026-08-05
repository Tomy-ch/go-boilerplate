package main

import (
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

func TestCollectVersions(t *testing.T) {
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
			require.Error(t, err)
			assert.Contains(t, err.Error(), "no version prefix")
		})
	})
}

func TestReportDuplicates(t *testing.T) {
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

func TestReportGaps(t *testing.T) {
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

func TestExpectedSequence(t *testing.T) {
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
}
