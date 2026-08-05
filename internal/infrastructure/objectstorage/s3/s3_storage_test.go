package s3_test

import (
	"context"
	"io"
	"net/http"          //nolint:depguard // 送出されたリクエストヘッダを記録するテスト専用のラッパに必要（本番 infra は http を直接使わない）
	"net/http/httptest" //nolint:depguard // gofakes3 の in-process S3 サーバ起動にテスト専用の httptest が必要（本番 infra は http を直接使わない）
	"net/textproto"     //nolint:depguard // ヘッダ名の正規化にテスト専用で必要（本番 infra は http を直接使わない）
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

// set は、受け取ったヘッダのスナップショットを記録します。
// 後続のリクエストで元のヘッダが再利用されても記録が変質しないよう複製します。
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

// has は、name のヘッダが送出されたかを返します。
// get は「送出されていない」と「空値で送出された」をどちらも空文字として返すため、
// 送出そのものの有無を問う場合はこちらを使います。
func (r *requestHeaders) has(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.h[textproto.CanonicalMIMEHeaderKey(name)]
	return ok
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

	return newStorageWithOutbound(t, endpoint, bucket, observability.NewDisabledOutboundHTTPClient(true))
}

// newStorageWithOutbound は、SSRF ガードの方針を指定して S3 実装を組み立てます。
func newStorageWithOutbound(
	t *testing.T, endpoint, bucket string, outbound *observability.OutboundHTTPClient,
) boundary.Storage {
	t.Helper()

	return s3adapter.New(s3adapter.Config{
		Endpoint:        endpoint,
		Region:          "us-east-1",
		Bucket:          bucket,
		AccessKeyID:     "test",
		SecretAccessKey: "test",
		UsePathStyle:    true,
		HTTPClient:      outbound,
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

		t.Run("渡した HTTPClient を SDK が実際に使う", func(t *testing.T) {
			t.Parallel()
			// fake は loopback（private）で待つため、private 網を許可した client でのみ到達できる。
			// ガード付き client が SDK へ届いていなければ、拒否側でも素通りして成功してしまう。
			ts, _, _ := newFakeS3(t)

			allowed := newStorageWithOutbound(
				t, ts.URL, testBucket, observability.NewDisabledOutboundHTTPClient(true))
			_, allowedErr := allowed.Put(context.Background(), boundary.PutObject{
				Key: "guard-allowed.txt", Body: []byte("x"), ContentType: "text/plain",
			})

			denied := newStorageWithOutbound(
				t, ts.URL, testBucket, observability.NewDisabledOutboundHTTPClient(false))
			_, deniedErr := denied.Put(context.Background(), boundary.PutObject{
				Key: "guard-denied.txt", Body: []byte("x"), ContentType: "text/plain",
			})

			require.NoError(t, allowedErr)
			require.Error(t, deniedErr, "private 網を拒否する client でも到達できており、SDK がガードを使っていない")
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
			assert.False(t, received.has("Cache-Control"))
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

func Test_storage_List(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("接頭辞に一致するオブジェクトをキーと更新時刻で返す", func(t *testing.T) {
			t.Parallel()

			ts, _, _ := newFakeS3(t)
			s := newStorage(t, ts.URL, testBucket)
			putObjects(t, s, "products/a.png", "products/b.png", "invoices/c.pdf")

			got, err := s.List(context.Background(), boundary.ListQuery{Prefix: "products/"})

			require.NoError(t, err)
			assert.Equal(t, []string{"products/a.png", "products/b.png"}, listedKeys(got))
			for _, o := range got.Objects {
				assert.False(t, o.ModifiedAt.IsZero())
			}
		})

		t.Run("接頭辞が一致しないオブジェクトは返さない", func(t *testing.T) {
			t.Parallel()

			ts, _, _ := newFakeS3(t)
			s := newStorage(t, ts.URL, testBucket)
			putObjects(t, s, "invoices/c.pdf")

			got, err := s.List(context.Background(), boundary.ListQuery{Prefix: "products/"})

			require.NoError(t, err)
			assert.Empty(t, got.Objects)
		})

		t.Run("Limit を超える場合は次カーソルを返し続きを取得できる", func(t *testing.T) {
			t.Parallel()

			ts, _, _ := newFakeS3(t)
			s := newStorage(t, ts.URL, testBucket)
			putObjects(t, s, "products/a.png", "products/b.png", "products/c.png")

			first, err := s.List(context.Background(), boundary.ListQuery{Prefix: "products/", Limit: 2})
			require.NoError(t, err)
			require.NotEmpty(t, first.NextCursor)
			assert.Len(t, first.Objects, 2)

			second, err := s.List(context.Background(),
				boundary.ListQuery{Prefix: "products/", Cursor: first.NextCursor, Limit: 2})
			require.NoError(t, err)

			assert.Empty(t, second.NextCursor)
			assert.Equal(t, []string{"products/a.png", "products/b.png", "products/c.png"},
				append(listedKeys(first), listedKeys(second)...))
		})

		t.Run("最終ページでは次カーソルを空で返す", func(t *testing.T) {
			t.Parallel()

			ts, _, _ := newFakeS3(t)
			s := newStorage(t, ts.URL, testBucket)
			putObjects(t, s, "products/a.png")

			got, err := s.List(context.Background(), boundary.ListQuery{Prefix: "products/", Limit: 10})

			require.NoError(t, err)
			assert.Empty(t, got.NextCursor)
		})

		t.Run("対象が無ければ空を返す", func(t *testing.T) {
			t.Parallel()

			ts, _, _ := newFakeS3(t)
			s := newStorage(t, ts.URL, testBucket)

			got, err := s.List(context.Background(), boundary.ListQuery{Prefix: "products/"})

			require.NoError(t, err)
			assert.Empty(t, got.Objects)
			assert.Empty(t, got.NextCursor)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("存在しないバケットの列挙は ErrUnavailable へ正規化する", func(t *testing.T) {
			t.Parallel()

			ts, _, _ := newFakeS3(t)
			s := newStorage(t, ts.URL, "missing-bucket")

			_, err := s.List(context.Background(), boundary.ListQuery{Prefix: "products/"})

			require.Error(t, err)
			assert.True(t, xerrors.Is(err, apperror.ErrUnavailable))
		})
	})
}

func Test_storage_Delete(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("指定したキーのオブジェクトだけを削除する", func(t *testing.T) {
			t.Parallel()

			ts, _, _ := newFakeS3(t)
			s := newStorage(t, ts.URL, testBucket)
			putObjects(t, s, "products/gone.png", "products/kept.png")

			require.NoError(t, s.Delete(context.Background(), []string{"products/gone.png"}))

			got, err := s.List(context.Background(), boundary.ListQuery{Prefix: "products/"})
			require.NoError(t, err)
			assert.Equal(t, []string{"products/kept.png"}, listedKeys(got))
		})

		t.Run("存在しないキーの削除は成功として扱い再実行しても結果が変わらない", func(t *testing.T) {
			t.Parallel()

			// 冪等性はこのジョブの再実行安全性そのもの。存在しないキーで失敗すると、
			// 一度回収された孤児が次回以降ジョブ全体を落とすようになる。
			ts, _, _ := newFakeS3(t)
			s := newStorage(t, ts.URL, testBucket)
			putObjects(t, s, "products/a.png")

			require.NoError(t, s.Delete(context.Background(), []string{"products/a.png"}))
			require.NoError(t, s.Delete(context.Background(), []string{"products/a.png"}))

			got, err := s.List(context.Background(), boundary.ListQuery{Prefix: "products/"})
			require.NoError(t, err)
			assert.Empty(t, got.Objects)
		})

		t.Run("キーが空なら何もせず成功する", func(t *testing.T) {
			t.Parallel()

			ts, _, _ := newFakeS3(t)
			s := newStorage(t, ts.URL, testBucket)
			putObjects(t, s, "products/a.png")

			require.NoError(t, s.Delete(context.Background(), nil))

			got, err := s.List(context.Background(), boundary.ListQuery{Prefix: "products/"})
			require.NoError(t, err)
			assert.Equal(t, []string{"products/a.png"}, listedKeys(got))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("存在しないバケットへの削除は ErrUnavailable へ正規化する", func(t *testing.T) {
			t.Parallel()

			ts, _, _ := newFakeS3(t)
			s := newStorage(t, ts.URL, "missing-bucket")

			err := s.Delete(context.Background(), []string{"products/a.png"})

			require.Error(t, err)
			assert.True(t, xerrors.Is(err, apperror.ErrUnavailable))
		})

		t.Run("一部のキーだけ削除に失敗した応答は ErrUnavailable として扱う", func(t *testing.T) {
			t.Parallel()

			// S3 はキー単位の失敗を呼び出しエラーではなく応答本文で返す。これを成功として扱うと、
			// 消えていないオブジェクトを消えたものとして数え、回収ジョブが以後その孤児を報告しなくなる。
			ts := newPartialDeleteFailureS3(t)
			s := newStorage(t, ts.URL, testBucket)

			err := s.Delete(context.Background(), []string{"products/ok.png", "products/ng.png"})

			require.Error(t, err)
			assert.True(t, xerrors.Is(err, apperror.ErrUnavailable))
		})
	})
}

// newPartialDeleteFailureS3 は、DeleteObjects に対してキー単位の失敗を含む応答を返す
// テスト専用の S3 サーバを起動します。gofakes3 のインメモリ実装は部分失敗を作れないため、
// 応答本文を直接組み立てます。
func newPartialDeleteFailureS3(t *testing.T) *httptest.Server {
	t.Helper()

	const body = `<?xml version="1.0" encoding="UTF-8"?>` +
		`<DeleteResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">` +
		`<Error><Key>products/ng.png</Key><Code>AccessDenied</Code><Message>Access Denied</Message></Error>` +
		`</DeleteResult>`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(ts.Close)
	return ts
}

// putObjects は、指定キーのオブジェクトを保存するテストヘルパーです。
func putObjects(t *testing.T, s boundary.Storage, keys ...string) {
	t.Helper()

	for _, k := range keys {
		_, err := s.Put(context.Background(), boundary.PutObject{
			Key:         k,
			Body:        []byte("data"),
			ContentType: "application/octet-stream",
		})
		require.NoError(t, err)
	}
}

// listedKeys は、列挙結果からキーだけを取り出すテストヘルパーです。
func listedKeys(r boundary.ListResult) []string {
	keys := make([]string, 0, len(r.Objects))
	for _, o := range r.Objects {
		keys = append(keys, o.Key)
	}
	return keys
}

// readAll は、io.Reader を最後まで読み切って返すテストヘルパーです。
func readAll(t *testing.T, r io.Reader) []byte {
	t.Helper()

	b, err := io.ReadAll(r)
	require.NoError(t, err)
	return b
}
