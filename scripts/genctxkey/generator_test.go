package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateCtxKey(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ctxキーとテストのファイルを出力する", func(t *testing.T) {
			t.Parallel()
			outDir := t.TempDir()
			require.NoError(t, GenerateCtxKey("request id", "string", "", "", outDir, ""))

			entries, err := os.ReadDir(outDir)
			require.NoError(t, err)
			assert.NotEmpty(t, entries)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("nameが空の場合、必須項目エラーを返す", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, GenerateCtxKey("", "string", "", "", t.TempDir(), ""), errMissingNameOrType)
		})

		t.Run("typeが空の場合、必須項目エラーを返す", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, GenerateCtxKey("request id", "", "", "", t.TempDir(), ""), errMissingNameOrType)
		})
	})
}

func Test_toExportedName(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("区切り文字で分割した各語を先頭大文字で連結する", func(t *testing.T) {
			t.Parallel()
			got, err := toExportedName("request-id value")
			require.NoError(t, err)
			assert.Equal(t, "RequestIdValue", got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("語が1つも取れない場合、不正な名前エラーを返す", func(t *testing.T) {
			t.Parallel()
			_, err := toExportedName("---")
			require.ErrorIs(t, err, errInvalidName)
		})

		t.Run("数字始まりでGoの識別子にならない場合、不正な識別子エラーを返す", func(t *testing.T) {
			t.Parallel()
			_, err := toExportedName("1st")
			require.ErrorIs(t, err, errInvalidIdentifier)
		})
	})
}

func Test_toIdentifierLower(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("区切り文字で分割した各語を小文字で連結する", func(t *testing.T) {
			t.Parallel()
			got, err := toIdentifierLower("Request-ID")
			require.NoError(t, err)
			assert.Equal(t, "requestid", got)
		})

		t.Run("数字始まりの場合、先頭にxを補って識別子にする", func(t *testing.T) {
			t.Parallel()
			got, err := toIdentifierLower("1st")
			require.NoError(t, err)
			assert.Equal(t, "x1st", got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("語が1つも取れない場合、不正な名前エラーを返す", func(t *testing.T) {
			t.Parallel()
			_, err := toIdentifierLower("---")
			require.ErrorIs(t, err, errInvalidName)
		})
	})
}

func Test_resolveImportAlias(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("importPathが空の場合、指定されたaliasをそのまま返す", func(t *testing.T) {
			t.Parallel()
			got, err := resolveImportAlias("uuid.UUID", "", "given")
			require.NoError(t, err)
			assert.Equal(t, "given", got)
		})

		t.Run("alias未指定で型に修飾子がある場合、修飾子を採用する", func(t *testing.T) {
			t.Parallel()
			got, err := resolveImportAlias("uuid.UUID", "github.com/google/uuid", "")
			require.NoError(t, err)
			assert.Equal(t, "uuid", got)
		})

		t.Run("alias未指定で型に修飾子がない場合、importPathの末尾を採用する", func(t *testing.T) {
			t.Parallel()
			got, err := resolveImportAlias("UUID", "github.com/google/uuid", "")
			require.NoError(t, err)
			assert.Equal(t, "uuid", got)
		})

		t.Run("修飾子とaliasが一致する場合、aliasを返す", func(t *testing.T) {
			t.Parallel()
			got, err := resolveImportAlias("uuid.UUID", "github.com/google/uuid", "uuid")
			require.NoError(t, err)
			assert.Equal(t, "uuid", got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("修飾子とaliasが食い違う場合、不一致エラーを返す", func(t *testing.T) {
			t.Parallel()
			_, err := resolveImportAlias("uuid.UUID", "github.com/google/uuid", "other")
			require.ErrorIs(t, err, errQualifierAliasMismatch)
			require.ErrorContains(t, err, "qualifier=uuid alias=other")
		})
	})
}

func Test_lastSegment(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("スラッシュ区切りの末尾要素を返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "uuid", lastSegment("github.com/google/uuid"))
		})

		t.Run("区切りが無い場合、入力をそのまま返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "uuid", lastSegment("uuid"))
		})
	})
}
