package inquiry

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	domaininquiry "go-boilerplate/internal/domain/inquiry"
	mock_inquiry "go-boilerplate/internal/domain/inquiry/mock"
	domainmessage "go-boilerplate/internal/domain/inquirymessage"
	mock_inquirymessage "go-boilerplate/internal/domain/inquirymessage/mock"
	"go-boilerplate/internal/observability"
	mock_authz "go-boilerplate/internal/usecase/boundary/authz/mock"
	clocktestkit "go-boilerplate/internal/usecase/boundary/clock/testkit"
	mock_realtime "go-boilerplate/internal/usecase/boundary/realtime/mock"
	"go-boilerplate/internal/usecase/boundary/tx"
	mock_tx "go-boilerplate/internal/usecase/boundary/tx/mock"
	mock_outbox "go-boilerplate/internal/usecase/outbox/mock"
	mock_ucrealtime "go-boilerplate/internal/usecase/realtime/mock"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"
)

// baseTime は、テストで用いる基準時刻です。
var baseTime = time.Date(2026, time.September, 1, 10, 0, 0, 0, time.UTC)

// deps は、テストで差し替える依存の束です。
type deps struct {
	repo      *mock_inquiry.MockRepository
	messages  *mock_inquirymessage.MockRepository
	sequences *mock_realtime.MockSequenceAllocator
	emit      *mock_outbox.MockEmitUsecase
	tickets   *mock_ucrealtime.MockTicketIssuer
	authz     *mock_authz.MockAuthorizer
}

// newTestUsecase は、すべての依存をモックにしたユースケースを組み立てます。
func newTestUsecase(t *testing.T) (*usecase, deps) {
	t.Helper()
	ctrl := gomock.NewController(t)
	d := deps{
		repo:      mock_inquiry.NewMockRepository(ctrl),
		messages:  mock_inquirymessage.NewMockRepository(ctrl),
		sequences: mock_realtime.NewMockSequenceAllocator(ctrl),
		emit:      mock_outbox.NewMockEmitUsecase(ctrl),
		tickets:   mock_ucrealtime.NewMockTicketIssuer(ctrl),
		authz:     mock_authz.NewMockAuthorizer(ctrl),
	}

	return &usecase{
		txm:        newPassthroughTx(t),
		clock:      clocktestkit.NewMockClock(t, baseTime),
		repo:       d.repo,
		msgRepo:    d.messages,
		sequences:  d.sequences,
		emit:       d.emit,
		tickets:    d.tickets,
		authorizer: d.authz,
		tracer:     observability.NewMockUsecaseLayerTracer(t),
	}, d
}

// newPassthroughTx は、渡された関数をそのまま実行する tx マネージャを組み立てます。
func newPassthroughTx(t *testing.T) tx.Manager {
	t.Helper()
	m := mock_tx.NewMockManager(gomock.NewController(t))
	m.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) },
	).AnyTimes()
	return m
}

// newTestInquiry は、永続化済みの問い合わせを組み立てます。
func newTestInquiry(t *testing.T, userID uuid.UUID) *domaininquiry.Inquiry {
	t.Helper()
	i, err := domaininquiry.Reconstruct(uuidtestkit.NewTestFromSalt(t, "inquiry"), domaininquiry.Attributes{
		UserID:    userID,
		CreatedAt: baseTime.Add(-time.Hour),
		UpdatedAt: baseTime.Add(-time.Hour),
	})
	require.NoError(t, err)
	return i
}

// newTestMessage は、指定した位置の永続化済みメッセージを組み立てます。
func newTestMessage(t *testing.T, inquiryID uuid.UUID, kind domainmessage.AuthorKind, sequence int64) *domainmessage.Message {
	t.Helper()
	author, err := domainmessage.NewAuthor(kind, uuidtestkit.NewTestFromSalt(t, "subject"))
	require.NoError(t, err)
	m, err := domainmessage.Reconstruct(uuidtestkit.NewTestFromSalt(t, "message"), domainmessage.Attributes{
		InquiryID: inquiryID,
		Author:    author,
		Body:      "本文",
		Sequence:  sequence,
		CreatedAt: baseTime,
	})
	require.NoError(t, err)
	return m
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("依存を渡すと非nilのユースケースを生成する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			uc := New(
				mock_tx.NewMockManager(ctrl),
				clocktestkit.NewMockClock(t, baseTime),
				mock_inquiry.NewMockRepository(ctrl),
				mock_inquirymessage.NewMockRepository(ctrl),
				mock_realtime.NewMockSequenceAllocator(ctrl),
				mock_outbox.NewMockEmitUsecase(ctrl),
				mock_ucrealtime.NewMockTicketIssuer(ctrl),
				mock_authz.NewMockAuthorizer(ctrl),
				observability.NewNoopTracerFactory(t),
			)

			assert.NotNil(t, uc)
		})
	})
}

func Test_toMessageView(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("集約の値を出力DTOへ写す", func(t *testing.T) {
			t.Parallel()
			inquiryID := uuidtestkit.NewTestFromSalt(t, "inquiry")
			m := newTestMessage(t, inquiryID, domainmessage.AuthorKindOperator, 7)

			view := toMessageView(m)

			assert.Equal(t, m.ID(), view.ID)
			assert.Equal(t, inquiryID, view.InquiryID)
			assert.Equal(t, "operator", view.AuthorKind)
			assert.Equal(t, "本文", view.Body)
			assert.Equal(t, int64(7), view.Sequence)
			assert.Equal(t, baseTime, view.CreatedAt)
		})
	})
}
