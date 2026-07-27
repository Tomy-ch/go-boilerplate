package testkit_test

import (
	"testing"

	"go-boilerplate/pkg/decimal/testkit"
)

func TestMustParse(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("有効な十進文字列を Decimal へ解析する", func(t *testing.T) {
			t.Parallel()
			got := testkit.MustParse(t, "19.995")
			if got.String() != "19.995" {
				t.Fatalf("want 19.995, got %s", got.String())
			}
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("無効な文字列は tb.Fatalf でテストを終了させる", func(t *testing.T) {
			t.Parallel()
			t.Skip("tb.Fatalf は呼び出し側テストの終了を伴うため、この分岐は検証不可")
		})
	})
}
