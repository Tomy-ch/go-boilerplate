package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/httpstack/oapi/validator"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	ucrealtime "go-boilerplate/internal/usecase/realtime"
	mock_realtime "go-boilerplate/internal/usecase/realtime/mock"
)

var apiKeyScheme = &openapi3.SecurityScheme{Type: "apiKey", In: "query", Name: "ticket"}

func newInput(t *testing.T, target string, scheme *openapi3.SecurityScheme, withSlot bool) *openapi3filter.AuthenticationInput {
	t.Helper()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, target, nil)
	if withSlot {
		req = req.WithContext(ctxhelper.WithStreamGrant(req.Context()))
	}
	return &openapi3filter.AuthenticationInput{
		RequestValidationInput: &openapi3filter.RequestValidationInput{
			Request:    req,
			PathParams: map[string]string{"destination": "stream-1"},
		},
		SecuritySchemeName: SchemeName,
		SecurityScheme:     scheme,
	}
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("担当するschemeはspecに宣言されたStreamTicketである", func(t *testing.T) {
			t.Parallel()
			spec, err := validator.GetValidator()
			require.NoError(t, err)
			declared, ok := spec.Components.SecuritySchemes[SchemeName]
			require.True(t, ok, "openapi.yaml の securitySchemes に %s が無い", SchemeName)
			assert.Equal(t, "apiKey", declared.Value.Type)
			assert.Equal(t, "query", declared.Value.In)

			s := New(mock_realtime.NewMockTicketVerifier(gomock.NewController(t)))
			assert.Equal(t, SchemeName, s.Scheme())
		})
	})
}

func Test_streamTicket_Scheme(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("StreamTicketを返す", func(t *testing.T) {
			t.Parallel()
			s := &streamTicket{}
			assert.Equal(t, "StreamTicket", s.Scheme())
		})
	})
}

func Test_streamTicket_Authenticate(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("宣言されたパラメータの生値とpathのdestinationで検証しStreamGrantへ書く", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			v := mock_realtime.NewMockTicketVerifier(ctrl)
			want := ucrealtime.VerifiedTicketView{Subject: "subject-1", Destination: "stream-1", Scope: "read", InitialCursor: 3}
			v.EXPECT().Verify(gomock.Any(), "raw-ticket", rt.StreamID("stream-1")).Return(want, nil)

			in := newInput(t, "/v1/streams/stream-1?ticket=raw-ticket&after=1", apiKeyScheme, true)
			require.NoError(t, New(v).Authenticate(context.Background(), in))

			got, ok := ctxhelper.GetStreamGrant(in.RequestValidationInput.Request.Context())
			assert.True(t, ok)
			assert.Equal(t, want, got)
		})

		t.Run("検証にはバリデータの引数ではなくリクエストのcontextを渡す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			v := mock_realtime.NewMockTicketVerifier(ctrl)
			type ctxKey struct{}
			in := newInput(t, "/v1/streams/stream-1?ticket=raw-ticket", apiKeyScheme, true)
			req := in.RequestValidationInput.Request
			in.RequestValidationInput.Request = req.WithContext(context.WithValue(req.Context(), ctxKey{}, "request"))
			v.EXPECT().Verify(gomock.Any(), "raw-ticket", rt.StreamID("stream-1")).
				DoAndReturn(func(ctx context.Context, _ string, _ rt.StreamID) (ucrealtime.VerifiedTicketView, error) {
					assert.Equal(t, "request", ctx.Value(ctxKey{}))
					return ucrealtime.VerifiedTicketView{}, nil
				})

			require.NoError(t, New(v).Authenticate(context.WithValue(context.Background(), ctxKey{}, "validator"), in))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("検証に失敗すればそのエラーを返しStreamGrantは書かれない", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			v := mock_realtime.NewMockTicketVerifier(ctrl)
			v.EXPECT().Verify(gomock.Any(), "", rt.StreamID("stream-1")).Return(ucrealtime.VerifiedTicketView{}, ucrealtime.ErrTicketInvalid)

			in := newInput(t, "/v1/streams/stream-1", apiKeyScheme, true)
			err := New(v).Authenticate(context.Background(), in)

			require.ErrorIs(t, err, ucrealtime.ErrTicketInvalid)
			_, ok := ctxhelper.GetStreamGrant(in.RequestValidationInput.Request.Context())
			assert.False(t, ok)
		})

		t.Run("schemeの宣言が渡されなければ検証せずErrSchemeDeclarationMissing", func(t *testing.T) {
			t.Parallel()
			v := mock_realtime.NewMockTicketVerifier(gomock.NewController(t))

			err := New(v).Authenticate(context.Background(), newInput(t, "/v1/streams/stream-1?ticket=raw", nil, true))

			require.ErrorIs(t, err, ErrSchemeDeclarationMissing)
		})

		t.Run("StreamGrantスロットが無ければErrStreamGrantSlotNotFound", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			v := mock_realtime.NewMockTicketVerifier(ctrl)
			v.EXPECT().Verify(gomock.Any(), "raw", rt.StreamID("stream-1")).Return(ucrealtime.VerifiedTicketView{}, nil)

			err := New(v).Authenticate(context.Background(), newInput(t, "/v1/streams/stream-1?ticket=raw", apiKeyScheme, false))

			require.ErrorIs(t, err, ErrStreamGrantSlotNotFound)
		})
	})
}
