package fnmeta

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_splitFuncName(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("フルパス付きメソッド名を分割できる", func(t *testing.T) {
			t.Parallel()
			full := "github.com/org/proj/internal/usecase/user.(*UserUsecase).GetUser"
			lhs, rhs := splitFuncName(full)
			assert.Equal(t, "user.(*UserUsecase)", lhs)
			assert.Equal(t, "GetUser", rhs)
		})

		t.Run("スラッシュのみがある場合は最後の要素をlhsにする", func(t *testing.T) {
			t.Parallel()
			full := "github.com/org/proj/pkg"
			lhs, rhs := splitFuncName(full)
			assert.Equal(t, "pkg", lhs)
			assert.Empty(t, rhs)
		})

		t.Run("ドットが無い場合は全体をlhsにする", func(t *testing.T) {
			t.Parallel()
			full := "main"
			lhs, rhs := splitFuncName(full)
			assert.Equal(t, "main", lhs)
			assert.Empty(t, rhs)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("空文字はunknownを返す", func(t *testing.T) {
			t.Parallel()
			lhs, rhs := splitFuncName("")
			assert.Equal(t, "unknown", lhs)
			assert.Empty(t, rhs)
		})
	})
}

func TestExtractFunctionName(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("メソッド名を抽出できる", func(t *testing.T) {
			t.Parallel()
			full := "github.com/org/proj/internal/usecase/user.(*UserUsecase).GetUser"
			assert.Equal(t, "GetUser", ExtractFunctionName(full))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("空文字はunknownを返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "unknown", ExtractFunctionName(""))
		})

		t.Run("ドットを含まない不正入力はunknownを返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "unknown", ExtractFunctionName("no_dot_here"))
		})
	})
}

func TestExtractPackageName(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ポインタレシーバのフルパスからパッケージ名を抽出できる", func(t *testing.T) {
			t.Parallel()
			full := "github.com/org/proj/internal/usecase/user.(*UserUsecase).GetUser"
			assert.Equal(t, "user", ExtractPackageName(full))
		})

		t.Run("値レシーバのフルパスからパッケージ名を抽出できる", func(t *testing.T) {
			t.Parallel()
			full := "github.com/org/proj/internal/usecase/user.UserUsecase.GetUser"
			assert.Equal(t, "user", ExtractPackageName(full))
		})

		t.Run("レシーバ無しの関数のフルパスからパッケージ名を抽出できる", func(t *testing.T) {
			t.Parallel()
			full := "github.com/org/proj/pkg.Func"
			assert.Equal(t, "pkg", ExtractPackageName(full))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("空文字はunknownを返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "unknown", ExtractPackageName(""))
		})

		t.Run("ドットを含まない不正入力はunknownを返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "unknown", ExtractPackageName("no_dot_here"))
		})
	})
}
