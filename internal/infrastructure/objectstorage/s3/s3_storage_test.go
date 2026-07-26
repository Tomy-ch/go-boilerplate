package s3_test

import (
	"context"
	"io"
	"net/http"          //nolint:depguard // 送出されたリクエストヘッダを記録するテスト専用のラッパに必要（本番 infra は http を直接使わない）
	"net/http/httptest" //nolint:depguard // gofakes3 の in-process S3 サーバ起動にテスト専用の httptest が必要（本番 infra は http を直接使わない）
	"sync"
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

// requestHeaders は、fake S3 が受け取った最後のリクエストヘッダを保持します。
// gofakes3 は Cache-Control をオブジェクトの metadata へ保存しない（保存対象は X-Amz-* /
// Content-Type / Content-Disposition / Content-Encoding のみ）ため、送出ヘッダ側で検証します。
type requestHeaders struct {
	mu sync.Mutex
	h  http.Header
}

// set は、受け取ったヘッダを記録します。
func (r *requestHeaders) set(h http.Header) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.h = h.Clone()
}

// get は、記録したヘッダから name の値を返します。未設定なら空文字を返します。
func (r *requestHeaders) get(name string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.h.Get(name)
}

// newFakeS3 は、gofakes3 の in-process S3 サーバを起動し、testBucket を作成して
// サーバ・バックエンド（保存内容の検証用）・受信ヘッダの記録を返します。
func newFakeS3(t *testing.T) (*httptest.Server, gofakes3.Backend, *requestHeaders) {
	t.Helper()

	be := s3mem.New()
	received := &requestHeaders{}
	fake := gofakes3.New(be).Server()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.set(r.Header)
		fake.ServeHTTP(w, r)
	}))
	t.Cleanup(ts.Close)

	require.NoError(t, be.CreateBucket(testBucket))
	return ts, be, received
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

			ts, be, _ := newFakeS3(t)
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

		t.Run("CacheControl を指定すると Cache-Control ヘッダを送出する", func(t *testing.T) {
			t.Parallel()

			ts, _, received := newFakeS3(t)
			s := newStorage(t, ts.URL, testBucket)

			_, err := s.Put(context.Background(), boundary.PutObject{
				Key:          "products/cache.png",
				Body:         []byte("data"),
				ContentType:  "image/png",
				CacheControl: "public, max-age=31536000, immutable",
			})

			require.NoError(t, err)
			assert.Equal(t, "public, max-age=31536000, immutable", received.get("Cache-Control"))
		})

		t.Run("CacheControl が空なら Cache-Control ヘッダを送出しない", func(t *testing.T) {
			t.Parallel()

			ts, _, received := newFakeS3(t)
			s := newStorage(t, ts.URL, testBucket)

			_, err := s.Put(context.Background(), boundary.PutObject{
				Key:         "products/nocache.png",
				Body:        []byte("data"),
				ContentType: "image/png",
			})

			require.NoError(t, err)
			assert.Empty(t, received.get("Cache-Control"))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("存在しないバケットへの保存は ErrUnavailable へ正規化する", func(t *testing.T) {
			t.Parallel()

			ts, _, _ := newFakeS3(t)
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
