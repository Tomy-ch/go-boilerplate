package seed

import (
	"context"
	"testing"

	"go-boilerplate/internal/logging"
	mock_fs "go-boilerplate/pkg/fs/mock"
	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	testSeedFile      = objectSeedPlace + "/products/a.webp"
	testSeedKey       = "products/a.webp"
	testLocalEndpoint = "http://garage:3900"
)

// recordingPut は、保存を依頼されたオブジェクトを recorded へ記録する PutObjectFunc を返します。
func recordingPut(recorded *[]ObjectToPut) PutObjectFunc {
	return func(_ context.Context, obj ObjectToPut) error {
		*recorded = append(*recorded, obj)
		return nil
	}
}

// failingPut は、常に err を返す PutObjectFunc を返します。
func failingPut(err error) PutObjectFunc {
	return func(context.Context, ObjectToPut) error { return err }
}

func TestRunObjectSeed(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("エンドポイントが空ならファイル走査もせず何もしない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			fsys := mock_fs.NewMockFS(ctrl)
			var recorded []ObjectToPut

			err := RunObjectSeed(logging.NewTestLogger(t), fsys, "", recordingPut(&recorded))

			require.NoError(t, err)
			assert.Empty(t, recorded)
		})

		t.Run("対象が1件も無ければ何もしない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			fsys := mock_fs.NewMockFS(ctrl)
			fsys.EXPECT().Glob(objectSeedPlace+"/*/*").Return(nil, nil)
			var recorded []ObjectToPut

			err := RunObjectSeed(logging.NewTestLogger(t), fsys, testLocalEndpoint, recordingPut(&recorded))

			require.NoError(t, err)
			assert.Empty(t, recorded)
		})

		t.Run("走査したファイルを投入する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			fsys := mock_fs.NewMockFS(ctrl)
			var recorded []ObjectToPut

			fsys.EXPECT().Glob(objectSeedPlace+"/*/*").Return([]string{testSeedFile}, nil)
			fsys.EXPECT().ReadFile(testSeedFile).Return([]byte("webp"), nil)

			err := RunObjectSeed(logging.NewTestLogger(t), fsys, testLocalEndpoint, recordingPut(&recorded))

			require.NoError(t, err)
			require.Len(t, recorded, 1)
			assert.Equal(t, testSeedKey, recorded[0].Key)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ファイル走査の失敗を伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			fsys := mock_fs.NewMockFS(ctrl)
			globErr := xerrors.New("glob failed")
			fsys.EXPECT().Glob(gomock.Any()).Return(nil, globErr)
			var recorded []ObjectToPut

			err := RunObjectSeed(logging.NewTestLogger(t), fsys, testLocalEndpoint, recordingPut(&recorded))

			require.ErrorIs(t, err, globErr)
		})
	})
}

func Test_collectSeedObjects(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("対応拡張子のみを昇順で返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			fsys := mock_fs.NewMockFS(ctrl)
			fsys.EXPECT().Glob(gomock.Any()).Return([]string{
				objectSeedPlace + "/products/b.png",
				objectSeedPlace + "/products/a.webp",
			}, nil)

			files, err := collectSeedObjects(context.Background(), logging.NewTestLogger(t), fsys)

			require.NoError(t, err)
			assert.Equal(t, []string{
				objectSeedPlace + "/products/a.webp",
				objectSeedPlace + "/products/b.png",
			}, files)
		})

		t.Run("接頭辞ディレクトリが異なるファイルも同じく対象にする", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			fsys := mock_fs.NewMockFS(ctrl)
			fsys.EXPECT().Glob(gomock.Any()).Return([]string{objectSeedPlace + "/banners/hero.webp"}, nil)

			files, err := collectSeedObjects(context.Background(), logging.NewTestLogger(t), fsys)

			require.NoError(t, err)
			assert.Equal(t, []string{objectSeedPlace + "/banners/hero.webp"}, files)
		})

		t.Run("ドットファイルは除外する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			fsys := mock_fs.NewMockFS(ctrl)
			fsys.EXPECT().Glob(gomock.Any()).Return([]string{objectSeedPlace + "/products/.gitkeep"}, nil)

			files, err := collectSeedObjects(context.Background(), logging.NewTestLogger(t), fsys)

			require.NoError(t, err)
			assert.Empty(t, files)
		})

		t.Run("非対応の拡張子は除外する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			fsys := mock_fs.NewMockFS(ctrl)
			fsys.EXPECT().Glob(gomock.Any()).Return([]string{objectSeedPlace + "/products/note.txt"}, nil)

			files, err := collectSeedObjects(context.Background(), logging.NewTestLogger(t), fsys)

			require.NoError(t, err)
			assert.Empty(t, files)
		})

		t.Run("拡張子の大文字小文字は区別しない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			fsys := mock_fs.NewMockFS(ctrl)
			fsys.EXPECT().Glob(gomock.Any()).Return([]string{objectSeedPlace + "/products/a.WEBP"}, nil)

			files, err := collectSeedObjects(context.Background(), logging.NewTestLogger(t), fsys)

			require.NoError(t, err)
			assert.Equal(t, []string{objectSeedPlace + "/products/a.WEBP"}, files)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Globの失敗を伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			fsys := mock_fs.NewMockFS(ctrl)
			globErr := xerrors.New("glob failed")
			fsys.EXPECT().Glob(gomock.Any()).Return(nil, globErr)

			_, err := collectSeedObjects(context.Background(), logging.NewTestLogger(t), fsys)

			require.ErrorIs(t, err, globErr)
		})
	})
}

func Test_putSeedObjects(t *testing.T) {
	t.Parallel()

	other := objectSeedPlace + "/products/b.png"

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("全件を投入する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			fsys := mock_fs.NewMockFS(ctrl)
			var recorded []ObjectToPut

			fsys.EXPECT().ReadFile(gomock.Any()).Return([]byte("img"), nil).Times(2)

			err := putSeedObjects(context.Background(), logging.NewTestLogger(t), fsys,
				recordingPut(&recorded), []string{testSeedFile, other})

			require.NoError(t, err)
			assert.Len(t, recorded, 2)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("1件失敗しても残りを試みてエラーをまとめて返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			fsys := mock_fs.NewMockFS(ctrl)
			readErr := xerrors.New("read failed")
			var recorded []ObjectToPut

			fsys.EXPECT().ReadFile(testSeedFile).Return(nil, readErr)
			fsys.EXPECT().ReadFile(other).Return([]byte("img"), nil)

			err := putSeedObjects(context.Background(), logging.NewTestLogger(t), fsys,
				recordingPut(&recorded), []string{testSeedFile, other})

			require.ErrorIs(t, err, readErr)
			assert.Len(t, recorded, 1)
		})
	})
}

func Test_putSeedObject(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("seed配下の相対パスをキーにして保存する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			fsys := mock_fs.NewMockFS(ctrl)
			body := []byte("webp bytes")
			var recorded []ObjectToPut

			fsys.EXPECT().ReadFile(testSeedFile).Return(body, nil)

			err := putSeedObject(context.Background(), logging.NewTestLogger(t), fsys,
				recordingPut(&recorded), testSeedFile)

			require.NoError(t, err)
			require.Len(t, recorded, 1)
			assert.Equal(t, ObjectToPut{
				Key:         testSeedKey,
				Body:        body,
				ContentType: "image/webp",
			}, recorded[0])
		})

		t.Run("接頭辞ディレクトリはキーの構造として保たれる", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			fsys := mock_fs.NewMockFS(ctrl)
			banner := objectSeedPlace + "/banners/hero.png"
			var recorded []ObjectToPut

			fsys.EXPECT().ReadFile(banner).Return([]byte("png"), nil)

			err := putSeedObject(context.Background(), logging.NewTestLogger(t), fsys,
				recordingPut(&recorded), banner)

			require.NoError(t, err)
			require.Len(t, recorded, 1)
			assert.Equal(t, "banners/hero.png", recorded[0].Key)
			assert.Equal(t, "image/png", recorded[0].ContentType)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ファイル読み込みの失敗を伝播し保存しない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			fsys := mock_fs.NewMockFS(ctrl)
			readErr := xerrors.New("read failed")
			var recorded []ObjectToPut

			fsys.EXPECT().ReadFile(gomock.Any()).Return(nil, readErr)

			err := putSeedObject(context.Background(), logging.NewTestLogger(t), fsys,
				recordingPut(&recorded), testSeedFile)

			require.ErrorIs(t, err, readErr)
			assert.Empty(t, recorded)
		})

		t.Run("保存の失敗を伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			fsys := mock_fs.NewMockFS(ctrl)
			putErr := xerrors.New("put failed")

			fsys.EXPECT().ReadFile(gomock.Any()).Return([]byte("img"), nil)

			err := putSeedObject(context.Background(), logging.NewTestLogger(t), fsys,
				failingPut(putErr), testSeedFile)

			require.ErrorIs(t, err, putErr)
		})
	})
}
