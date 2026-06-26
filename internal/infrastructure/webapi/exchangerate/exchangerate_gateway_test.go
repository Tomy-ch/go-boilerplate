package exchangerate_test

import (
	"context"
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/infrastructure/httpclient"
	mock_httpclient "go-boilerplate/internal/infrastructure/httpclient/mock"
	"go-boilerplate/internal/infrastructure/webapi/exchangerate"
	"go-boilerplate/internal/observability"
	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const testEndpoint exchangerate.Endpoint = "https://api.exchangerate.example.com"

func TestGatewayGetRate(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Requestを正しく組み立て外部レスポンスをRate_DTOへ変換する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			client := mock_httpclient.NewMockClient(ctrl)
			client.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, req *httpclient.Request) (*httpclient.Response, error) {
					assert.Equal(t, httpclient.Downstream("exchangerate"), req.Downstream())
					assert.Equal(t, httpclient.MethodGet, req.Method())
					assert.Contains(t, req.URL(), "base=USD")
					assert.Contains(t, req.URL(), "quote=JPY")
					return &httpclient.Response{StatusCode: 200, Body: []byte(`{"rate":150.5}`)}, nil
				})

			gw := exchangerate.New(testEndpoint, client, observability.NewNoopTracerFactory(t))
			rate, err := gw.GetRate(context.Background(), "USD", "JPY")

			require.NoError(t, err)
			require.NotNil(t, rate)
			assert.Equal(t, "USD", rate.Base)
			assert.Equal(t, "JPY", rate.Quote)
			assert.InEpsilon(t, 150.5, rate.Value, 1e-9)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("substrateが返すapperrorをそのまま伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			client := mock_httpclient.NewMockClient(ctrl)
			client.EXPECT().Do(gomock.Any(), gomock.Any()).
				Return(nil, xerrors.Wrap(apperror.ErrNotFound, "downstream 404"))

			gw := exchangerate.New(testEndpoint, client, observability.NewNoopTracerFactory(t))
			_, err := gw.GetRate(context.Background(), "USD", "JPY")

			require.ErrorIs(t, err, apperror.ErrNotFound)
		})

		t.Run("不正なJSONレスポンスはErrUnavailableを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			client := mock_httpclient.NewMockClient(ctrl)
			client.EXPECT().Do(gomock.Any(), gomock.Any()).
				Return(&httpclient.Response{StatusCode: 200, Body: []byte(`not-json`)}, nil)

			gw := exchangerate.New(testEndpoint, client, observability.NewNoopTracerFactory(t))
			_, err := gw.GetRate(context.Background(), "USD", "JPY")

			require.ErrorIs(t, err, apperror.ErrUnavailable)
		})

		t.Run("rateが0以下のレスポンスはErrUnavailableを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			client := mock_httpclient.NewMockClient(ctrl)
			client.EXPECT().Do(gomock.Any(), gomock.Any()).
				Return(&httpclient.Response{StatusCode: 200, Body: []byte(`{"rate":0}`)}, nil)

			gw := exchangerate.New(testEndpoint, client, observability.NewNoopTracerFactory(t))
			_, err := gw.GetRate(context.Background(), "USD", "JPY")

			require.ErrorIs(t, err, apperror.ErrUnavailable)
		})
	})
}
