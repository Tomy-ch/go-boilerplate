package yamlblock_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"go-boilerplate/scripts/lib/yamlblock"
)

// lines は行の並びを 1 つの内容へ組み立てる。
func lines(rows ...string) string {
	return strings.Join(rows, "\n")
}

func Test_ContentLines(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("リテラルブロックの中身を行番号で返す", func(t *testing.T) {
			t.Parallel()
			got := yamlblock.ContentLines(lines("steps:", "  - run: |", "      echo one", "      echo two"))
			assert.Equal(t, map[int]bool{3: true, 4: true}, got)
		})

		t.Run("折りたたみブロックの中身も対象にする", func(t *testing.T) {
			t.Parallel()
			got := yamlblock.ContentLines(lines("steps:", "  - run: >-", "      echo one"))
			assert.Equal(t, map[int]bool{3: true}, got)
		})

		t.Run("chomp と字下げの指示子が付いたヘッダも解釈する", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, map[int]bool{3: true},
				yamlblock.ContentLines(lines("steps:", "  - run: |2-", "      echo one")))
			assert.Equal(t, map[int]bool{3: true},
				yamlblock.ContentLines(lines("steps:", "  - run: |-2", "      echo one")))
		})

		t.Run("字下げがヘッダ以下へ戻った行でブロックを終える", func(t *testing.T) {
			t.Parallel()
			got := yamlblock.ContentLines(lines("  - run: |", "      echo one", "  - uses: actions/checkout@v7"))
			assert.Equal(t, map[int]bool{2: true}, got)
		})

		t.Run("ブロックの途中の空行を中身として扱う", func(t *testing.T) {
			t.Parallel()
			got := yamlblock.ContentLines(lines("  - run: |", "      echo one", "", "      echo two"))
			assert.Equal(t, map[int]bool{2: true, 3: true, 4: true}, got)
		})

		t.Run("複数のブロックをそれぞれ独立に扱う", func(t *testing.T) {
			t.Parallel()
			got := yamlblock.ContentLines(lines(
				"  - run: |", "      echo one", "  - name: X", "  - run: |", "      echo two"))
			assert.Equal(t, map[int]bool{2: true, 5: true}, got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ヘッダ行そのものを中身に含めない", func(t *testing.T) {
			t.Parallel()
			got := yamlblock.ContentLines(lines("  - run: |", "      echo one"))
			assert.False(t, got[1])
		})

		t.Run("ブロックスカラーを持たない内容では空を返す", func(t *testing.T) {
			t.Parallel()
			got := yamlblock.ContentLines(lines("steps:", "  - uses: actions/checkout@v7"))
			assert.Empty(t, got)
		})

		t.Run("値が同じ行にある通常のスカラーをヘッダとして扱わない", func(t *testing.T) {
			t.Parallel()
			got := yamlblock.ContentLines(lines("  - run: echo one", "      継続に見える行"))
			assert.Empty(t, got)
		})
	})
}
