package realtime

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/observability"
	mock_clock "go-boilerplate/internal/usecase/boundary/clock/mock"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	mock_realtime "go-boilerplate/internal/usecase/boundary/realtime/mock"
	"go-boilerplate/pkg/xerrors"
)

const secretValue = "opaque-256-bit-value"

var errRandom = xerrors.New("random unavailable")

func newTicketServiceForTest(t *testing.T) (*ticketService, *mock_realtime.MockStreamTicketStore, *mock_realtime.MockSecretGenerator) {
	t.Helper()

	ctrl := gomock.NewController(t)
	store := mock_realtime.NewMockStreamTicketStore(ctrl)
	secrets := mock_realtime.NewMockSecretGenerator(ctrl)
	clk := mock_clock.NewMockClock(ctrl)
	clk.EXPECT().Now().Return(now).AnyTimes()

	return &ticketService{store: store, secrets: secrets, clock: clk, tracer: observability.NewNoopTracerFactory(t).Usecase()}, store, secrets
}

func savedTicket(destination rt.StreamID) rt.StreamTicket {
	return rt.StreamTicket{
		Hash: hashTicket(secretValue), Subject: "alice", Destination: destination, Scope: "read", InitialCursor: 7,
		IssuedAt: now, ExpiresAt: now.Add(TicketTTL),
	}
}

func TestNewTicketIssuer(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	issuer := NewTicketIssuer(mock_realtime.NewMockStreamTicketStore(ctrl), mock_realtime.NewMockSecretGenerator(ctrl),
		mock_clock.NewMockClock(ctrl), observability.NewNoopTracerFactory(t))
	assert.NotNil(t, issuer)
}

func TestNewTicketVerifier(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	verifier := NewTicketVerifier(mock_realtime.NewMockStreamTicketStore(ctrl), mock_clock.NewMockClock(ctrl), observability.NewNoopTracerFactory(t))
	assert.NotNil(t, verifier)
}

func Test_newTicketService(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	s := newTicketService(mock_realtime.NewMockStreamTicketStore(ctrl), nil, mock_clock.NewMockClock(ctrl), observability.NewNoopTracerFactory(t))
	assert.NotNil(t, s.store)
	assert.Nil(t, s.secrets, "検証だけなら乱数源は要らない")
}

func Test_ticketService_Issue(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("生値の hash を bindings と期限付きで保存し、生値と期限を返す", func(t *testing.T) {
			t.Parallel()

			s, store, secrets := newTicketServiceForTest(t)
			secrets.EXPECT().Generate().Return(secretValue, nil)
			store.EXPECT().Save(gomock.Any(), savedTicket("stream-a")).Return(nil)

			got, err := s.Issue(t.Context(), IssueTicketInput{Subject: "alice", Destination: "stream-a", Scope: "read", InitialCursor: 7})
			require.NoError(t, err)
			assert.Equal(t, IssuedTicket{Value: secretValue, ExpiresAt: now.Add(TicketTTL)}, got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("乱数源が使えなければ保存せずにエラーを返す", func(t *testing.T) {
			t.Parallel()

			s, _, secrets := newTicketServiceForTest(t)
			secrets.EXPECT().Generate().Return("", errRandom)

			_, err := s.Issue(t.Context(), IssueTicketInput{Subject: "alice", Destination: "stream-a"})
			require.ErrorIs(t, err, errRandom)
		})

		t.Run("保存に失敗すれば store のエラーをそのまま返す", func(t *testing.T) {
			t.Parallel()

			s, store, secrets := newTicketServiceForTest(t)
			secrets.EXPECT().Generate().Return(secretValue, nil)
			store.EXPECT().Save(gomock.Any(), gomock.Any()).Return(errStoreOff)

			_, err := s.Issue(t.Context(), IssueTicketInput{Subject: "alice", Destination: "stream-a"})
			require.ErrorIs(t, err, apperror.ErrUnavailable)
		})
	})
}

func Test_ticketService_Verify(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("hash が一致し destination も合えば ticket を返す", func(t *testing.T) {
			t.Parallel()

			s, store, _ := newTicketServiceForTest(t)
			store.EXPECT().Find(gomock.Any(), hashTicket(secretValue), now).Return(savedTicket("stream-a"), true, nil)

			got, err := s.Verify(t.Context(), secretValue, "stream-a")
			require.NoError(t, err)
			assert.Equal(t, savedTicket("stream-a"), got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("生値が空なら store を読まずに ErrTicketInvalid", func(t *testing.T) {
			t.Parallel()

			s, _, _ := newTicketServiceForTest(t)

			_, err := s.Verify(t.Context(), "", "stream-a")
			require.ErrorIs(t, err, ErrTicketInvalid)
			require.ErrorIs(t, err, apperror.ErrUnauthenticated)
		})

		t.Run("無い（または期限切れの）ticket は ErrTicketInvalid", func(t *testing.T) {
			t.Parallel()

			s, store, _ := newTicketServiceForTest(t)
			store.EXPECT().Find(gomock.Any(), hashTicket(secretValue), now).Return(rt.StreamTicket{}, false, nil)

			_, err := s.Verify(t.Context(), secretValue, "stream-a")
			require.ErrorIs(t, err, ErrTicketInvalid)
		})

		t.Run("destination が違えば ErrTicketInvalid", func(t *testing.T) {
			t.Parallel()

			s, store, _ := newTicketServiceForTest(t)
			store.EXPECT().Find(gomock.Any(), hashTicket(secretValue), now).Return(savedTicket("stream-a"), true, nil)

			_, err := s.Verify(t.Context(), secretValue, "stream-b")
			require.ErrorIs(t, err, ErrTicketInvalid)
		})

		t.Run("store が読めなければそのエラーをそのまま返す", func(t *testing.T) {
			t.Parallel()

			s, store, _ := newTicketServiceForTest(t)
			store.EXPECT().Find(gomock.Any(), hashTicket(secretValue), now).Return(rt.StreamTicket{}, false, errStoreOff)

			_, err := s.Verify(t.Context(), secretValue, "stream-a")
			require.ErrorIs(t, err, apperror.ErrUnavailable)
			require.NotErrorIs(t, err, ErrTicketInvalid)
		})
	})
}

func Test_hashTicket(t *testing.T) {
	t.Parallel()

	sum := sha256.Sum256([]byte(secretValue))
	assert.Equal(t, rt.TicketHash(hex.EncodeToString(sum[:])), hashTicket(secretValue))
	assert.NotEqual(t, hashTicket("a"), hashTicket("b"))
	assert.Len(t, hashTicket("a"), 64)
}
