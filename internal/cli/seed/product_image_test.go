package seed

import (
	"context"
	"testing"

	"go-boilerplate/internal/infrastructure/rdb/driver"
	mock_driver "go-boilerplate/internal/infrastructure/rdb/driver/mock"
	"go-boilerplate/internal/logging"
	mock_fs "go-boilerplate/pkg/fs/mock"
	"go-boilerplate/pkg/xerrors"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	testProductID     = "f211d877-36ef-41dc-aea0-72968a7d8f7e"
	testOtherID       = "5266f905-1af4-477a-bc7c-e69729e22a2b"
	testImagePath     = productImageSeedPlace + "/" + testProductID + ".webp"
	testImageKey      = productImageKeyPrefix + testProductID + ".webp"
	testLocalEndpoint = "http://garage:3900"
)

// errOpenDBNotExpected は、DB を開いてはならない経路で openDB が呼ばれたことを示すエラーです。
var errOpenDBNotExpected = xerrors.New("openDB must not be called")

// rejectOpenDB は、呼ばれること自体が失敗である openDB です。
func rejectOpenDB(logging.Logger, string) (driver.DatabaseDriver, error) {
	return nil, errOpenDBNotExpected
}

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

// updatedTag は、1 行更新された CommandTag を返します。
func updatedTag(t *testing.T) pgconn.CommandTag {
	t.Helper()

	return pgconn.NewCommandTag("UPDATE 1")
}

func TestRunProductImageSeed(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("エンドポイントが空ならファイル走査もDB接続もせず何もしない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			fsys := mock_fs.NewMockFS(ctrl)
			var recorded []ObjectToPut

			err := RunProductImageSeed(logging.NewTestLogger(t), fsys, "local", "", recordingPut(&recorded), rejectOpenDB)

			require.NoError(t, err)
			assert.Empty(t, recorded)
		})

		t.Run("画像が1枚も無ければDBに接続しない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			fsys := mock_fs.NewMockFS(ctrl)
			fsys.EXPECT().Glob(productImageSeedPlace+"/*").Return(nil, nil)
			var recorded []ObjectToPut

			err := RunProductImageSeed(logging.NewTestLogger(t), fsys, "local", testLocalEndpoint,
				recordingPut(&recorded), rejectOpenDB)

			require.NoError(t, err)
			assert.Empty(t, recorded)
		})

		t.Run("画像を保存しDB接続を閉じる", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			fsys := mock_fs.NewMockFS(ctrl)
			db := mock_driver.NewMockDatabaseDriver(ctrl)
			var recorded []ObjectToPut

			fsys.EXPECT().Glob(productImageSeedPlace+"/*").Return([]string{testImagePath}, nil)
			fsys.EXPECT().ReadFile(testImagePath).Return([]byte("webp"), nil)
			db.EXPECT().Exec(gomock.Any(), updateProductImagePathSQL, testImageKey, testProductID).
				Return(updatedTag(t), nil)
			db.EXPECT().Close().Return(nil)

			err := RunProductImageSeed(logging.NewTestLogger(t), fsys, "local", testLocalEndpoint,
				recordingPut(&recorded),
				func(logging.Logger, string) (driver.DatabaseDriver, error) { return db, nil })

			require.NoError(t, err)
			assert.Len(t, recorded, 1)
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

			err := RunProductImageSeed(logging.NewTestLogger(t), fsys, "local", testLocalEndpoint,
				recordingPut(&recorded), rejectOpenDB)

			require.ErrorIs(t, err, globErr)
		})

		t.Run("DB接続の失敗を伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			fsys := mock_fs.NewMockFS(ctrl)
			fsys.EXPECT().Glob(gomock.Any()).Return([]string{testImagePath}, nil)
			var recorded []ObjectToPut

			err := RunProductImageSeed(logging.NewTestLogger(t), fsys, "local", testLocalEndpoint,
				recordingPut(&recorded), rejectOpenDB)

			require.ErrorIs(t, err, errOpenDBNotExpected)
		})
	})
}

func Test_collectProductImageFiles(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("対応拡張子のみを昇順で返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			fsys := mock_fs.NewMockFS(ctrl)
			fsys.EXPECT().Glob(gomock.Any()).Return([]string{
				productImageSeedPlace + "/b.png",
				productImageSeedPlace + "/a.webp",
			}, nil)

			files, err := collectProductImageFiles(context.Background(), logging.NewTestLogger(t), fsys)

			require.NoError(t, err)
			assert.Equal(t, []string{
				productImageSeedPlace + "/a.webp",
				productImageSeedPlace + "/b.png",
			}, files)
		})

		t.Run("ドットファイルは除外する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			fsys := mock_fs.NewMockFS(ctrl)
			fsys.EXPECT().Glob(gomock.Any()).Return([]string{productImageSeedPlace + "/.gitkeep"}, nil)

			files, err := collectProductImageFiles(context.Background(), logging.NewTestLogger(t), fsys)

			require.NoError(t, err)
			assert.Empty(t, files)
		})

		t.Run("非対応の拡張子は除外する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			fsys := mock_fs.NewMockFS(ctrl)
			fsys.EXPECT().Glob(gomock.Any()).Return([]string{productImageSeedPlace + "/note.txt"}, nil)

			files, err := collectProductImageFiles(context.Background(), logging.NewTestLogger(t), fsys)

			require.NoError(t, err)
			assert.Empty(t, files)
		})

		t.Run("拡張子の大文字小文字は区別しない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			fsys := mock_fs.NewMockFS(ctrl)
			fsys.EXPECT().Glob(gomock.Any()).Return([]string{productImageSeedPlace + "/a.WEBP"}, nil)

			files, err := collectProductImageFiles(context.Background(), logging.NewTestLogger(t), fsys)

			require.NoError(t, err)
			assert.Equal(t, []string{productImageSeedPlace + "/a.WEBP"}, files)
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

			_, err := collectProductImageFiles(context.Background(), logging.NewTestLogger(t), fsys)

			require.ErrorIs(t, err, globErr)
		})
	})
}

func Test_seedProductImages(t *testing.T) {
	t.Parallel()

	otherPath := productImageSeedPlace + "/" + testOtherID + ".png"

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("全件を投入する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			fsys := mock_fs.NewMockFS(ctrl)
			db := mock_driver.NewMockDatabaseDriver(ctrl)
			var recorded []ObjectToPut

			fsys.EXPECT().ReadFile(gomock.Any()).Return([]byte("img"), nil).Times(2)
			db.EXPECT().Exec(gomock.Any(), updateProductImagePathSQL, gomock.Any(), gomock.Any()).
				Return(updatedTag(t), nil).Times(2)

			err := seedProductImages(context.Background(), logging.NewTestLogger(t), fsys, db,
				recordingPut(&recorded), []string{testImagePath, otherPath})

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
			db := mock_driver.NewMockDatabaseDriver(ctrl)
			readErr := xerrors.New("read failed")
			var recorded []ObjectToPut

			fsys.EXPECT().ReadFile(testImagePath).Return(nil, readErr)
			fsys.EXPECT().ReadFile(otherPath).Return([]byte("img"), nil)
			db.EXPECT().Exec(gomock.Any(), updateProductImagePathSQL, gomock.Any(), gomock.Any()).
				Return(updatedTag(t), nil)

			err := seedProductImages(context.Background(), logging.NewTestLogger(t), fsys, db,
				recordingPut(&recorded), []string{testImagePath, otherPath})

			require.ErrorIs(t, err, readErr)
			assert.Len(t, recorded, 1)
		})
	})
}

func Test_seedProductImage(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ファイル名からキーと商品IDを導き保存と更新を行う", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			fsys := mock_fs.NewMockFS(ctrl)
			db := mock_driver.NewMockDatabaseDriver(ctrl)
			body := []byte("webp bytes")
			var recorded []ObjectToPut

			fsys.EXPECT().ReadFile(testImagePath).Return(body, nil)
			db.EXPECT().Exec(gomock.Any(), updateProductImagePathSQL, testImageKey, testProductID).
				Return(updatedTag(t), nil)

			err := seedProductImage(context.Background(), logging.NewTestLogger(t), fsys, db,
				recordingPut(&recorded), testImagePath)

			require.NoError(t, err)
			require.Len(t, recorded, 1)
			assert.Equal(t, ObjectToPut{
				Key:          testImageKey,
				Body:         body,
				ContentType:  "image/webp",
				CacheControl: productImageCacheControl,
			}, recorded[0])
		})

		t.Run("該当する商品が無くてもエラーにしない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			fsys := mock_fs.NewMockFS(ctrl)
			db := mock_driver.NewMockDatabaseDriver(ctrl)
			var recorded []ObjectToPut

			fsys.EXPECT().ReadFile(gomock.Any()).Return([]byte("img"), nil)
			db.EXPECT().Exec(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(pgconn.NewCommandTag("UPDATE 0"), nil)

			err := seedProductImage(context.Background(), logging.NewTestLogger(t), fsys, db,
				recordingPut(&recorded), testImagePath)

			require.NoError(t, err)
		})

		t.Run("jpg拡張子はimage/jpegとして保存する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			fsys := mock_fs.NewMockFS(ctrl)
			db := mock_driver.NewMockDatabaseDriver(ctrl)
			jpgPath := productImageSeedPlace + "/" + testProductID + ".jpg"
			var recorded []ObjectToPut

			fsys.EXPECT().ReadFile(jpgPath).Return([]byte("jpg"), nil)
			db.EXPECT().Exec(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(updatedTag(t), nil)

			err := seedProductImage(context.Background(), logging.NewTestLogger(t), fsys, db,
				recordingPut(&recorded), jpgPath)

			require.NoError(t, err)
			require.Len(t, recorded, 1)
			assert.Equal(t, "image/jpeg", recorded[0].ContentType)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ファイル読み込みの失敗を伝播し保存しない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			fsys := mock_fs.NewMockFS(ctrl)
			db := mock_driver.NewMockDatabaseDriver(ctrl)
			readErr := xerrors.New("read failed")
			var recorded []ObjectToPut

			fsys.EXPECT().ReadFile(gomock.Any()).Return(nil, readErr)

			err := seedProductImage(context.Background(), logging.NewTestLogger(t), fsys, db,
				recordingPut(&recorded), testImagePath)

			require.ErrorIs(t, err, readErr)
			assert.Empty(t, recorded)
		})

		t.Run("保存の失敗を伝播しDBを更新しない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			fsys := mock_fs.NewMockFS(ctrl)
			db := mock_driver.NewMockDatabaseDriver(ctrl)
			putErr := xerrors.New("put failed")

			fsys.EXPECT().ReadFile(gomock.Any()).Return([]byte("img"), nil)

			err := seedProductImage(context.Background(), logging.NewTestLogger(t), fsys, db,
				failingPut(putErr), testImagePath)

			require.ErrorIs(t, err, putErr)
		})

		t.Run("image_path更新の失敗を伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			fsys := mock_fs.NewMockFS(ctrl)
			db := mock_driver.NewMockDatabaseDriver(ctrl)
			execErr := xerrors.New("exec failed")
			var recorded []ObjectToPut

			fsys.EXPECT().ReadFile(gomock.Any()).Return([]byte("img"), nil)
			db.EXPECT().Exec(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(pgconn.CommandTag{}, execErr)

			err := seedProductImage(context.Background(), logging.NewTestLogger(t), fsys, db,
				recordingPut(&recorded), testImagePath)

			require.ErrorIs(t, err, execErr)
		})
	})
}
