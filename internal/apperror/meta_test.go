package apperror_test

import (
	"fmt"
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// plainError は fmt.Formatter を実装しない素の error です。
// MetaError.Format のフォールバック分岐(委譲先が Formatter でない場合)を検証するために使います。
type plainError struct{ msg string }

func (e plainError) Error() string { return e.msg }

// asMetaError は、err のチェーンから *MetaError を取り出します（メソッド単体の契約を検証するため）。
func asMetaError(t *testing.T, err error) *apperror.MetaError {
	t.Helper()
	var metaErr *apperror.MetaError
	require.True(t, xerrors.As(err, &metaErr))
	return metaErr
}

func TestNewMeta(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Code と Details が設定され Message は空", func(t *testing.T) {
			t.Parallel()
			meta := apperror.NewMeta("CUSTOM_CODE", "firstName", "email")
			assert.Equal(t, "CUSTOM_CODE", meta.Code())
			assert.Empty(t, meta.Message())
			assert.Equal(t, []string{"firstName", "email"}, meta.Details())
		})

		t.Run("渡した Details スライスを後から変更しても構築済みメタに影響しない", func(t *testing.T) {
			t.Parallel()
			details := []string{"firstName"}
			meta := apperror.NewMeta("", details...)
			details[0] = "mutated"
			assert.Equal(t, []string{"firstName"}, meta.Details())
		})

		t.Run("Details 無しの場合 nil を返す", func(t *testing.T) {
			t.Parallel()
			meta := apperror.NewMeta("CUSTOM_CODE")
			assert.Nil(t, meta.Details())
		})
	})
}

func TestMeta_WithMessage(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Message が上書きされたコピーを返し元は変わらない", func(t *testing.T) {
			t.Parallel()
			base := apperror.NewMeta("CUSTOM_CODE", "firstName")
			overridden := base.WithMessage("custom message")
			assert.Equal(t, "custom message", overridden.Message())
			assert.Equal(t, "CUSTOM_CODE", overridden.Code())
			assert.Equal(t, []string{"firstName"}, overridden.Details())
			assert.Empty(t, base.Message())
		})
	})
}

func TestMeta_Details(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("返り値を書き換えても内部状態に影響しない", func(t *testing.T) {
			t.Parallel()
			meta := apperror.NewMeta("", "firstName")
			got := meta.Details()
			got[0] = "mutated"
			assert.Equal(t, []string{"firstName"}, meta.Details())
		})
	})
}

func TestWithMeta(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("元エラーのセンチネル分類を保持する", func(t *testing.T) {
			t.Parallel()
			err := apperror.WithMeta(xerrors.Wrap(apperror.ErrValidation, "invalid"), apperror.NewMeta("CUSTOM"))
			require.ErrorIs(t, err, apperror.ErrValidation)
		})

		t.Run("Join された複数センチネルの分類をすべて保持する", func(t *testing.T) {
			t.Parallel()
			joined := xerrors.Join(
				xerrors.Wrap(apperror.ErrValidation, "validation failed"),
				xerrors.Wrap(apperror.ErrNotFound, "missing"),
			)
			err := apperror.WithMeta(joined, apperror.NewMeta("CUSTOM"))
			require.ErrorIs(t, err, apperror.ErrValidation)
			require.ErrorIs(t, err, apperror.ErrNotFound)
		})

		t.Run("エラーメッセージは元エラーのまま", func(t *testing.T) {
			t.Parallel()
			err := apperror.WithMeta(xerrors.New("original"), apperror.NewMeta("CUSTOM"))
			assert.Equal(t, "original", err.Error())
		})

		t.Run("スタックトレース表現は元エラーへ委譲される", func(t *testing.T) {
			t.Parallel()
			err := apperror.WithMeta(xerrors.Wrap(apperror.ErrValidation, "invalid"), apperror.NewMeta(""))
			assert.Contains(t, xerrors.StackTrace(err), "meta_test.go")
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("nil の場合 nil を返す", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, apperror.WithMeta(nil, apperror.NewMeta("CUSTOM")))
		})
	})
}

func TestWithDetails(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Details のみが付与されセンチネル分類を保持する", func(t *testing.T) {
			t.Parallel()
			err := apperror.WithDetails(xerrors.Wrap(apperror.ErrValidation, "invalid"), "firstName", "email")
			require.ErrorIs(t, err, apperror.ErrValidation)
			meta, ok := apperror.MetaFrom(err)
			require.True(t, ok)
			assert.Empty(t, meta.Code())
			assert.Empty(t, meta.Message())
			assert.Equal(t, []string{"firstName", "email"}, meta.Details())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("nil の場合 nil を返す", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, apperror.WithDetails(nil, "firstName"))
		})
	})
}

func TestMetaFrom(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("付与した Code / Message / Details を抽出できる", func(t *testing.T) {
			t.Parallel()
			err := apperror.WithMeta(apperror.ErrValidation,
				apperror.NewMeta("CUSTOM_CODE", "firstName", "email").WithMessage("custom message"))
			meta, ok := apperror.MetaFrom(err)
			require.True(t, ok)
			assert.Equal(t, "CUSTOM_CODE", meta.Code())
			assert.Equal(t, "custom message", meta.Message())
			assert.Equal(t, []string{"firstName", "email"}, meta.Details())
		})

		t.Run("さらにラップされていても抽出できる", func(t *testing.T) {
			t.Parallel()
			err := xerrors.Wrap(apperror.WithDetails(apperror.ErrValidation, "email"), "update failed")
			meta, ok := apperror.MetaFrom(err)
			require.True(t, ok)
			assert.Equal(t, []string{"email"}, meta.Details())
		})

		t.Run("多重に付与されている場合は外側が勝つ", func(t *testing.T) {
			t.Parallel()
			inner := apperror.WithMeta(apperror.ErrValidation, apperror.NewMeta("INNER"))
			outer := apperror.WithMeta(inner, apperror.NewMeta("OUTER"))
			meta, ok := apperror.MetaFrom(outer)
			require.True(t, ok)
			assert.Equal(t, "OUTER", meta.Code())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("メタ無しエラーの場合 ok=false", func(t *testing.T) {
			t.Parallel()
			_, ok := apperror.MetaFrom(xerrors.Wrap(apperror.ErrValidation, "invalid"))
			assert.False(t, ok)
		})

		t.Run("nil の場合 ok=false", func(t *testing.T) {
			t.Parallel()
			_, ok := apperror.MetaFrom(nil)
			assert.False(t, ok)
		})
	})
}

func TestMetaError_Format(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("元エラーがfmt.Formatterの場合_スタックトレース付き表現へ委譲する", func(t *testing.T) {
			t.Parallel()
			err := apperror.WithMeta(xerrors.Wrap(apperror.ErrValidation, "boom"), apperror.NewMeta("CODE"))

			out := fmt.Sprintf("%+v", err)
			assert.Contains(t, out, "boom")
			// %+v は委譲先 xerrors のスタックトレース(このテスト関数名)を含む
			assert.Contains(t, out, "TestMetaError_Format")
		})

		t.Run("元エラーがfmt.Formatterでない場合_Error文字列にフォールバックする", func(t *testing.T) {
			t.Parallel()
			err := apperror.WithMeta(plainError{msg: "plain"}, apperror.NewMeta("CODE"))

			assert.Equal(t, "plain", fmt.Sprintf("%+v", err))
		})
	})
}

func TestMetaError_Error(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Meta の利用者向け文言ではなく元エラーのメッセージを返す", func(t *testing.T) {
			t.Parallel()
			err := apperror.WithMeta(xerrors.New("original failure"),
				apperror.NewMeta("CUSTOM_CODE", "firstName").WithMessage("利用者向け文言"))

			metaErr := asMetaError(t, err)

			assert.Equal(t, "original failure", metaErr.Error())
		})
	})
}

func TestMetaError_Meta(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("付与した Code / Message / Details をそのまま返す", func(t *testing.T) {
			t.Parallel()
			err := apperror.WithMeta(apperror.ErrValidation,
				apperror.NewMeta("CUSTOM_CODE", "firstName", "email").WithMessage("custom message"))

			meta := asMetaError(t, err).Meta()

			assert.Equal(t, "CUSTOM_CODE", meta.Code())
			assert.Equal(t, "custom message", meta.Message())
			assert.Equal(t, []string{"firstName", "email"}, meta.Details())
		})

		t.Run("返した Meta の Details を書き換えても内部状態に影響しない", func(t *testing.T) {
			t.Parallel()
			err := apperror.WithMeta(apperror.ErrValidation, apperror.NewMeta("CUSTOM_CODE", "firstName"))
			metaErr := asMetaError(t, err)

			got := metaErr.Meta().Details()
			got[0] = "mutated"

			assert.Equal(t, []string{"firstName"}, metaErr.Meta().Details())
		})
	})
}

func TestMetaError_Unwrap(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("センチネル分類を保ったままメタ付与前のエラーを返す", func(t *testing.T) {
			t.Parallel()
			err := apperror.WithMeta(xerrors.Wrap(apperror.ErrValidation, "invalid"), apperror.NewMeta("CUSTOM_CODE"))

			unwrapped := asMetaError(t, err).Unwrap()

			require.ErrorIs(t, unwrapped, apperror.ErrValidation)
			_, ok := apperror.MetaFrom(unwrapped)
			assert.False(t, ok)
		})

		t.Run("多重付与ではメタ層を 1 段だけ外し内側の Meta が残る", func(t *testing.T) {
			t.Parallel()
			inner := apperror.WithMeta(apperror.ErrValidation, apperror.NewMeta("INNER"))
			outer := apperror.WithMeta(inner, apperror.NewMeta("OUTER"))

			unwrapped := asMetaError(t, outer).Unwrap()

			meta, ok := apperror.MetaFrom(unwrapped)
			require.True(t, ok)
			assert.Equal(t, "INNER", meta.Code())
		})
	})
}

func TestMeta_Code(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("NewMeta に渡したコードを返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "CUSTOM_CODE", apperror.NewMeta("CUSTOM_CODE", "firstName").Code())
		})

		t.Run("コード未指定の場合は空文字を返す", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, apperror.NewMeta("", "firstName").Code())
		})
	})
}

func TestMeta_Message(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("WithMessage で設定した文言を返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "custom message", apperror.NewMeta("CUSTOM_CODE").WithMessage("custom message").Message())
		})

		t.Run("WithMessage 未適用の場合は空文字を返す", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, apperror.NewMeta("CUSTOM_CODE", "firstName").Message())
		})
	})
}
