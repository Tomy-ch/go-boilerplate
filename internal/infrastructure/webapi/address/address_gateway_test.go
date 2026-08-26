package address_test

import (
	"context"
	"strings"
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/infrastructure/httpclient"
	mock_httpclient "go-boilerplate/internal/infrastructure/httpclient/mock"
	"go-boilerplate/internal/infrastructure/webapi/address"
	"go-boilerplate/internal/observability"
	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const testEndpoint address.Endpoint = "https://zipcloud.example.com"

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Endpoint_Client_TracerFactory から Gateway を生成する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			client := mock_httpclient.NewMockClient(ctrl)

			gw := address.New(testEndpoint, client, observability.NewNoopTracerFactory(t))

			assert.NotNil(t, gw)
		})
	})
}

func Test_gateway_Lookup(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Requestを正しく組み立て外部レスポンスをCandidate_DTOへ変換する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			client := mock_httpclient.NewMockClient(ctrl)
			client.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, req *httpclient.Request) (*httpclient.Response, error) {
					assert.Equal(t, httpclient.Downstream("address"), req.Downstream())
					assert.Equal(t, httpclient.MethodGet(), req.Method())
					// endpoint / パス / クエリまで含めて組み立てを検証する（endpoint 欠落や別ホスト固定の退行を捕捉）。
					assert.Equal(t, string(testEndpoint)+"/api/search?zipcode=1000001", req.URL())
					return &httpclient.Response{StatusCode: 200, Body: []byte(
						`{"status":200,"message":null,"results":[` +
							`{"address1":"東京都","address2":"千代田区","address3":"千代田"},` +
							`{"address1":"東京都","address2":"千代田区","address3":"大手町"}]}`,
					)}, nil
				})

			gw := address.New(testEndpoint, client, observability.NewNoopTracerFactory(t))
			candidates, err := gw.Lookup(context.Background(), "1000001")

			require.NoError(t, err)
			require.Len(t, candidates, 2)
			assert.Equal(t, "東京都", candidates[0].PrefectureName)
			assert.Equal(t, "千代田区", candidates[0].City)
			assert.Equal(t, "千代田", candidates[0].Town)
			assert.Equal(t, "大手町", candidates[1].Town)
		})

		t.Run("該当なし_results_nullは空スライスとnilエラーを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			client := mock_httpclient.NewMockClient(ctrl)
			client.EXPECT().Do(gomock.Any(), gomock.Any()).
				Return(&httpclient.Response{StatusCode: 200, Body: []byte(`{"status":200,"message":null,"results":null}`)}, nil)

			gw := address.New(testEndpoint, client, observability.NewNoopTracerFactory(t))
			candidates, err := gw.Lookup(context.Background(), "0000000")

			require.NoError(t, err)
			assert.Empty(t, candidates)
		})

		t.Run("候補が上限を超える応答は上限件数へ切り詰める", func(t *testing.T) {
			t.Parallel()

			// 上限（50）+1 件を返し、防御的に 50 件へ切り詰められることを検証する。
			one := `{"address1":"東京都","address2":"千代田区","address3":"千代田"}`
			body := `{"status":200,"results":[` + strings.Repeat(one+",", 50) + one + `]}`

			ctrl := gomock.NewController(t)
			client := mock_httpclient.NewMockClient(ctrl)
			client.EXPECT().Do(gomock.Any(), gomock.Any()).
				Return(&httpclient.Response{StatusCode: 200, Body: []byte(body)}, nil)

			gw := address.New(testEndpoint, client, observability.NewNoopTracerFactory(t))
			candidates, err := gw.Lookup(context.Background(), "1000001")

			require.NoError(t, err)
			assert.Len(t, candidates, 50)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("substrateが返すapperrorをそのまま伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			client := mock_httpclient.NewMockClient(ctrl)
			client.EXPECT().Do(gomock.Any(), gomock.Any()).
				Return(nil, xerrors.Wrap(apperror.ErrUnavailable, "downstream unavailable"))

			gw := address.New(testEndpoint, client, observability.NewNoopTracerFactory(t))
			candidates, err := gw.Lookup(context.Background(), "1000001")

			require.ErrorIs(t, err, apperror.ErrUnavailable)
			assert.Nil(t, candidates)
		})

		t.Run("不正なJSONレスポンスはErrUnavailableを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			client := mock_httpclient.NewMockClient(ctrl)
			client.EXPECT().Do(gomock.Any(), gomock.Any()).
				Return(&httpclient.Response{StatusCode: 200, Body: []byte(`not-json`)}, nil)

			gw := address.New(testEndpoint, client, observability.NewNoopTracerFactory(t))
			candidates, err := gw.Lookup(context.Background(), "1000001")

			require.ErrorIs(t, err, apperror.ErrUnavailable)
			assert.Nil(t, candidates)
		})

		t.Run("body_statusが200以外のレスポンスはErrUnavailableを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			client := mock_httpclient.NewMockClient(ctrl)
			client.EXPECT().Do(gomock.Any(), gomock.Any()).
				Return(&httpclient.Response{StatusCode: 200, Body: []byte(`{"status":400,"message":"パラメータが不正です。","results":null}`)}, nil)

			gw := address.New(testEndpoint, client, observability.NewNoopTracerFactory(t))
			candidates, err := gw.Lookup(context.Background(), "1000001")

			require.ErrorIs(t, err, apperror.ErrUnavailable)
			assert.Nil(t, candidates)
		})

		t.Run("body_statusが200以外でmessageがnullでもErrUnavailableを返す", func(t *testing.T) {
			t.Parallel()

			// message 欠落（null）でもエラーメッセージ組み立てで panic せず ErrUnavailable へ写像すること。
			ctrl := gomock.NewController(t)
			client := mock_httpclient.NewMockClient(ctrl)
			client.EXPECT().Do(gomock.Any(), gomock.Any()).
				Return(&httpclient.Response{StatusCode: 200, Body: []byte(`{"status":500,"results":null}`)}, nil)

			gw := address.New(testEndpoint, client, observability.NewNoopTracerFactory(t))
			candidates, err := gw.Lookup(context.Background(), "1000001")

			require.ErrorIs(t, err, apperror.ErrUnavailable)
			assert.Nil(t, candidates)
		})
	})
}
