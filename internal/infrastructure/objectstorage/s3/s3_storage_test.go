package s3_test

import (
	"context"
	"io"
	"net/http/httptest" //nolint:depguard // gofakes3 の in-process S3 サーバ起動にテスト専用の httptest が必要（本番 infra は http を直接使わない）
	"testing"

	"github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3mem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/internal/apperror"
	s3adapter "go-boilerplate/internal/infrastructure/objectstorage/s3"
	"go-boilerplate/internal/observability"
	boundary "go-boilerplate/internal/usecase/boundary/objectstorage"
	"go-boilerplate/pkg/xerrors"
)

const testBucket = "test-bucket"

// newFakeS3 は、gofakes3 の in-process S3 サーバを起動し、testBucket を作成して
// サーバとバックエンド（保存内容の検証用）を返します。
func newFakeS3(t *testing.T) (*httptest.Server, gofakes3.Backend) {
	t.Helper()

	be := s3mem.New()
	ts := httptest.NewServer(gofakes3.New(be).Server())
	t.Cleanup(ts.Close)

	require.NoError(t, be.CreateBucket(testBucket))
	return ts, be
}

// newStorage は、指定エンドポイント・バケットの fake S3 に接続した boundary.Storage を生成します。
func newStorage(t *testing.T, endpoint, bucket string) boundary.Storage {
	t.Helper()

	return s3adapter.New(s3adapter.Config{
		Endpoint:        endpoint,
		Region:          "us-east-1",
		Bucket:          bucket,
		AccessKeyID:     "test",
		SecretAccessKey: "test",
		UsePathStyle:    true,
	}, observability.NewNoopTracerFactory(t))
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Config_TracerFactory から Storage を生成する", func(t *testing.T) {
			t.Parallel()

			s := newStorage(t, "http://s3.example.com", testBucket)

			assert.NotNil(t, s)
		})
	})
}

func Test_storage_Put(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("オブジェクトを保存し保存したキーを Path として返す", func(t *testing.T) {
			t.Parallel()

			ts, be := newFakeS3(t)
			s := newStorage(t, ts.URL, testBucket)
			body := []byte("\x89PNG\r\n\x1a\n fake image bytes")

			path, err := s.Put(context.Background(), boundary.PutObject{
				Key:         "products/abc123.png",
				Body:        body,
				ContentType: "image/png",
			})

			require.NoError(t, err)
			assert.Equal(t, boundary.Path("products/abc123.png"), path)

			obj, gerr := be.GetObject(testBucket, "products/abc123.png", nil)
			require.NoError(t, gerr)
			defer func() { _ = obj.Contents.Close() }()
			assert.Equal(t, body, readAll(t, obj.Contents))
			assert.Equal(t, "image/png", obj.Metadata["Content-Type"])
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("存在しないバケットへの保存は ErrUnavailable へ正規化する", func(t *testing.T) {
			t.Parallel()

			ts, _ := newFakeS3(t)
			s := newStorage(t, ts.URL, "missing-bucket")

			_, err := s.Put(context.Background(), boundary.PutObject{
				Key:         "products/x.png",
				Body:        []byte("data"),
				ContentType: "image/png",
			})

			require.Error(t, err)
			assert.True(t, xerrors.Is(err, apperror.ErrUnavailable))
		})
	})
}

// readAll は、io.Reader を最後まで読み切って返すテストヘルパーです。
func readAll(t *testing.T, r io.Reader) []byte {
	t.Helper()

	b, err := io.ReadAll(r)
	require.NoError(t, err)
	return b
}
