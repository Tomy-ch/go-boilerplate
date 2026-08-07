package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const constTpl = "package sample\n\nconst  {{.NameLower}}  =  \"{{.NameCamel}}\"\n"

func TestGenerateCtxKey(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ctxキーとテストのファイルを名前由来のファイル名で出力する", func(t *testing.T) {
			t.Parallel()
			outDir := t.TempDir()
			require.NoError(t, GenerateCtxKey("request id", "string", "", "", outDir, ""))

			entries, err := os.ReadDir(outDir)
			require.NoError(t, err)
			require.Len(t, entries, 2)
			assert.Equal(t, "requestid_ctx.gen.go", entries[0].Name())
			assert.Equal(t, "requestid_ctx_test.go", entries[1].Name())
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

		t.Run("小文字識別子を組み立てられない名前の場合、小文字側を示す不正な識別子エラーを返す", func(t *testing.T) {
			t.Parallel()
			outDir := t.TempDir()

			err := GenerateCtxKey("a½", "string", "", "", outDir, "")
			require.ErrorIs(t, err, errInvalidIdentifier)
			require.ErrorContains(t, err, "a½")

			entries, readErr := os.ReadDir(outDir)
			require.NoError(t, readErr)
			assert.Empty(t, entries)
		})

		t.Run("公開識別子を組み立てられない名前の場合、不正な識別子エラーを返す", func(t *testing.T) {
			t.Parallel()
			outDir := t.TempDir()
			require.ErrorIs(t, GenerateCtxKey("1st", "string", "", "", outDir, ""), errInvalidIdentifier)

			entries, err := os.ReadDir(outDir)
			require.NoError(t, err)
			assert.Empty(t, entries)
		})

		t.Run("型の修飾子とaliasが食い違う場合、不一致エラーを返す", func(t *testing.T) {
			t.Parallel()
			err := GenerateCtxKey("request id", "uuid.UUID", "github.com/google/uuid", "other", t.TempDir(), "")
			require.ErrorIs(t, err, errQualifierAliasMismatch)
		})

		t.Run("出力先ディレクトリを作れない場合、mkdirエラーを返す", func(t *testing.T) {
			t.Parallel()
			blocker := filepath.Join(t.TempDir(), "blocker")
			require.NoError(t, os.WriteFile(blocker, []byte("not a directory"), 0o600))

			outDir := filepath.Join(blocker, "sub")
			err := GenerateCtxKey("request id", "string", "", "", outDir, "")
			require.Error(t, err)
			require.ErrorContains(t, err, "mkdir "+outDir)
		})

		t.Run("ctxキーのコードを整形できない型の場合、どちらのファイルも作らずエラーを返す", func(t *testing.T) {
			t.Parallel()
			outDir := t.TempDir()
			err := GenerateCtxKey("request id", "[]", "", "", outDir, "")
			require.Error(t, err)
			require.ErrorContains(t, err, "requestid_ctx.gen.go")

			entries, readErr := os.ReadDir(outDir)
			require.NoError(t, readErr)
			assert.Empty(t, entries)
		})

		t.Run("テストコードだけ整形できない場合、テストファイルを作らずエラーを返す", func(t *testing.T) {
			t.Parallel()
			outDir := t.TempDir()
			err := GenerateCtxKey("request id", "error", "", "", outDir, "!!!")
			require.Error(t, err)
			require.ErrorContains(t, err, "requestid_ctx_test.go")

			_, statErr := os.Stat(filepath.Join(outDir, "requestid_ctx.gen.go"))
			require.NoError(t, statErr)
			_, statErr = os.Stat(filepath.Join(outDir, "requestid_ctx_test.go"))
			require.ErrorIs(t, statErr, os.ErrNotExist)
		})
	})
}

func Test_resolveOutDir(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("空文字はカレントディレクトリを指す", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, ".", resolveOutDir(""))
		})

		t.Run("冗長な区切りと相対要素を畳んだパスを返す", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, filepath.Join("a", "c"), resolveOutDir("a//b/../c/"))
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

		t.Run("英数字でない数値文字が残る場合、不正な識別子エラーを返す", func(t *testing.T) {
			t.Parallel()
			_, err := toIdentifierLower("a½")
			require.ErrorIs(t, err, errInvalidIdentifier)
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

func Test_writeFile(t *testing.T) {
	t.Parallel()

	p := Param{NameLower: "requestid", NameCamel: "RequestId"}
	const formatted = "package sample\n\nconst requestid = \"RequestId\"\n"

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("テンプレートの展開結果をgofmt済みのGoコードとして書き出す", func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "out.go")
			require.NoError(t, writeFile(path, constTpl, p))

			got, err := os.ReadFile(path) //nolint:gosec // テストが自分で作った一時ファイル
			require.NoError(t, err)
			assert.Equal(t, formatted, string(got))
		})

		t.Run("既存の内容と一致する場合、書き直さない", func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "out.go")
			require.NoError(t, writeFile(path, constTpl, p))

			past := time.Now().Add(-time.Hour)
			require.NoError(t, os.Chtimes(path, past, past))
			require.NoError(t, writeFile(path, constTpl, p))

			info, err := os.Stat(path)
			require.NoError(t, err)
			assert.WithinDuration(t, past, info.ModTime(), time.Second)
		})

		t.Run("既存の内容と異なる場合、新しい内容で書き直す", func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "out.go")
			require.NoError(t, writeFile(path, constTpl, p))
			require.NoError(t, writeFile(path, constTpl, Param{NameLower: "traceid", NameCamel: "TraceId"}))

			got, err := os.ReadFile(path) //nolint:gosec // テストが自分で作った一時ファイル
			require.NoError(t, err)
			assert.Equal(t, "package sample\n\nconst traceid = \"TraceId\"\n", string(got))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("テンプレートの展開に失敗した場合、ファイルを作らずエラーを返す", func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "out.go")
			err := writeFile(path, "package sample\n\n// {{.Missing}}\n", p)
			require.Error(t, err)
			require.ErrorContains(t, err, "execute template "+path)

			_, statErr := os.Stat(path)
			require.ErrorIs(t, statErr, os.ErrNotExist)
		})

		t.Run("展開結果がGoコードとして整形できない場合、ファイルを作らずエラーを返す", func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "out.go")
			err := writeFile(path, "package sample\n\nfunc {{.NameLower}} {\n", p)
			require.Error(t, err)
			require.ErrorContains(t, err, "format "+path)

			_, statErr := os.Stat(path)
			require.ErrorIs(t, statErr, os.ErrNotExist)
		})

		t.Run("書き出し先がディレクトリの場合、writeエラーを返す", func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "out.go")
			require.NoError(t, os.Mkdir(path, 0o750))

			err := writeFile(path, constTpl, p)
			require.Error(t, err)
			require.ErrorContains(t, err, "write "+path)
		})
	})
}

func Test_splitWords(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("英数字以外を区切りとして語に分割する", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, []string{"request", "id", "value"}, splitWords("request-id value"))
		})

		t.Run("先頭と末尾に区切りがあっても空の語を作らない", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, []string{"request", "id"}, splitWords("--request__id--"))
		})

		t.Run("区切りを含まない場合、1語として返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, []string{"requestid"}, splitWords("requestid"))
		})

		t.Run("ASCII以外の文字は区切りとみなさず1語として保つ", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, []string{"リクエストID"}, splitWords("リクエストID"))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("区切り文字だけの場合、語を1つも返さない", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, splitWords("---"))
		})

		t.Run("空文字の場合、語を1つも返さない", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, splitWords(""))
		})
	})
}

func Test_isValidIdentifier(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("英字始まりで英数字が続く場合、識別子として扱う", func(t *testing.T) {
			t.Parallel()
			assert.True(t, isValidIdentifier("requestID2"))
		})

		t.Run("アンダースコア始まりの場合、識別子として扱う", func(t *testing.T) {
			t.Parallel()
			assert.True(t, isValidIdentifier("_x"))
		})

		t.Run("ASCII以外の文字も識別子として扱う", func(t *testing.T) {
			t.Parallel()
			assert.True(t, isValidIdentifier("リクエスト"))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("空文字の場合、識別子として扱わない", func(t *testing.T) {
			t.Parallel()
			assert.False(t, isValidIdentifier(""))
		})

		t.Run("数字始まりの場合、識別子として扱わない", func(t *testing.T) {
			t.Parallel()
			assert.False(t, isValidIdentifier("1st"))
		})

		t.Run("ハイフンを含む場合、識別子として扱わない", func(t *testing.T) {
			t.Parallel()
			assert.False(t, isValidIdentifier("a-b"))
		})

		t.Run("ドットを含む場合、識別子として扱わない", func(t *testing.T) {
			t.Parallel()
			assert.False(t, isValidIdentifier("a.b"))
		})
	})
}

func Test_sanitizeAlias(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ハイフンをアンダースコアへ置き換える", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "go_uuid", sanitizeAlias("go-uuid"))
		})

		t.Run("ドットをアンダースコアへ置き換える", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "v2_pkg", sanitizeAlias("v2.pkg"))
		})

		t.Run("そのまま識別子になる名前は変えない", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "uuid", sanitizeAlias("uuid"))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("置き換えても識別子にならない場合、pkgへ退避する", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "pkg", sanitizeAlias("2uuid"))
		})

		t.Run("Goの予約語になる場合、pkgへ退避する", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "pkg", sanitizeAlias("type"))
		})

		t.Run("空文字の場合、pkgへ退避する", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "pkg", sanitizeAlias(""))
		})
	})
}

func Test_extractQualifier(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("修飾子付きの型から修飾子を返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "uuid", extractQualifier("uuid.UUID"))
		})

		t.Run("修飾子が複数ある場合、最後のものを返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "uuid", extractQualifier("map[time.Time]uuid.UUID"))
		})

		t.Run("スライスやポインタが付いていても修飾子を返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "uuid", extractQualifier("[]*uuid.UUID"))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("修飾子が無い型の場合、空文字を返す", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, extractQualifier("UUID"))
		})

		t.Run("空文字の場合、空文字を返す", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, extractQualifier(""))
		})
	})
}

func Test_resolveTestValue(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("string型の場合、名前を含む成功値と空文字の失敗値を返す", func(t *testing.T) {
			t.Parallel()
			success, fail := resolveTestValue("string", "requestid", "")
			assert.Equal(t, `"test-requestid"`, success)
			assert.Equal(t, `""`, fail)
		})

		t.Run("int型の場合、非ゼロの成功値とゼロの失敗値を返す", func(t *testing.T) {
			t.Parallel()
			success, fail := resolveTestValue("int", "count", "")
			assert.Equal(t, "123", success)
			assert.Equal(t, "0", fail)
		})

		t.Run("bool型の場合、trueとfalseを返す", func(t *testing.T) {
			t.Parallel()
			success, fail := resolveTestValue("bool", "enabled", "")
			assert.Equal(t, "true", success)
			assert.Equal(t, "false", fail)
		})

		t.Run("組み込み型の場合、overrideを無視して既定値を採る", func(t *testing.T) {
			t.Parallel()
			success, fail := resolveTestValue("string", "requestid", "custom()")
			assert.Equal(t, `"test-requestid"`, success)
			assert.Equal(t, `""`, fail)
		})

		t.Run("任意型でoverrideがある場合、成功値に採用しゼロ値を失敗値にする", func(t *testing.T) {
			t.Parallel()
			success, fail := resolveTestValue("uuid.UUID", "userid", "uuid.New()")
			assert.Equal(t, "uuid.New()", success)
			assert.Equal(t, "*new(uuid.UUID)", fail)
		})

		t.Run("任意型でoverrideが無い場合、成功値も失敗値もゼロ値にする", func(t *testing.T) {
			t.Parallel()
			success, fail := resolveTestValue("uuid.UUID", "userid", "")
			assert.Equal(t, "*new(uuid.UUID)", success)
			assert.Equal(t, "*new(uuid.UUID)", fail)
		})
	})
}
