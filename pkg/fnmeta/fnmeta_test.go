package fnmeta

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSplitFuncName(t *testing.T) {
	t.Parallel()
	t.Run("空文字は unknown を返す", func(t *testing.T) {
		t.Parallel()
		lhs, rhs := splitFuncName("")
		require.Equal(t, "unknown", lhs)
		require.Empty(t, rhs)
	})

	t.Run("フルパス付きメソッド名を分割できる", func(t *testing.T) {
		t.Parallel()
		full := "github.com/org/proj/internal/usecase/user.(*UserUsecase).GetUser"
		lhs, rhs := splitFuncName(full)
		require.Equal(t, "user.(*UserUsecase)", lhs)
		require.Equal(t, "GetUser", rhs)
	})

	t.Run("スラッシュのみがある場合は最後の要素を lhs にする", func(t *testing.T) {
		t.Parallel()
		full := "github.com/org/proj/pkg"
		lhs, rhs := splitFuncName(full)
		require.Equal(t, "pkg", lhs)
		require.Empty(t, rhs)
	})

	t.Run("ドットが無い場合は全体を lhs にする", func(t *testing.T) {
		t.Parallel()
		full := "main"
		lhs, rhs := splitFuncName(full)
		require.Equal(t, "main", lhs)
		require.Empty(t, rhs)
	})
}

func TestExtractFunctionName(t *testing.T) {
	t.Parallel()
	t.Run("メソッド名を抽出できる", func(t *testing.T) {
		t.Parallel()
		full := "github.com/org/proj/internal/usecase/user.(*UserUsecase).GetUser"
		require.Equal(t, "GetUser", ExtractFunctionName(full))
	})

	t.Run("関数名が得られない場合は unknown を返す", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, "unknown", ExtractFunctionName(""))
		require.Equal(t, "unknown", ExtractFunctionName("no_dot_here"))
	})
}

func TestExtractPackageName(t *testing.T) {
	t.Parallel()
	t.Run("パッケージ名を抽出できる (レシーバあり)", func(t *testing.T) {
		t.Parallel()
		full := "github.com/org/proj/internal/usecase/user.(*UserUsecase).GetUser"
		require.Equal(t, "user", ExtractPackageName(full))
	})

	t.Run("パッケージ名を抽出できる (通常のケース)", func(t *testing.T) {
		t.Parallel()
		full := "github.com/org/proj/pkg.Func"
		require.Equal(t, "pkg", ExtractPackageName(full))
	})

	t.Run("空文字は unknown を返す", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, "unknown", ExtractPackageName(""))
	})
}
