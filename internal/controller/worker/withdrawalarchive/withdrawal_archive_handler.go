package withdrawalarchive

import (
	"context"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	workerbd "go-boilerplate/internal/usecase/boundary/worker"
	"go-boilerplate/internal/usecase/user"
	"go-boilerplate/internal/usecase/user/event"
	"go-boilerplate/pkg/xerrors"
)

// skippedMessage は、処理対象外のイベントを読み飛ばしたときのログメッセージです。
const skippedMessage = "skipped a message of another event type"

var _ workerbd.Handler = (*handler)(nil)

// handler は、退会イベントを退会証跡の保存へ写す worker.Handler です。
type handler struct {
	archive user.ArchiveUsecase
	tracer  observability.LayerTracer
	logging logging.Logger
}

// newHandler は、退会証跡 Handler を初期化します。
func newHandler(archive user.ArchiveUsecase, tf observability.TracerFactory, logger logging.Logger) *handler {
	return &handler{
		archive: archive,
		tracer:  tf.Controller(),
		logging: logger,
	}
}

// Handle は、user.withdrawn.v1 の payload を退会証跡として保存します。
// 種別が一致しないメッセージと種別属性を持たないメッセージは保存せず成功を返します。
// 復元できない payload は Permanent、保存の失敗は分類せずそのまま返します。
// これらの選別・分類の根拠は README（Message selection / Error classification）を参照。
func (h *handler) Handle(ctx context.Context, m workerbd.Message) error {
	ctx, endSpan := h.tracer.Start(ctx)
	defer endSpan()

	if m.Attributes[workerbd.AttrEventType] != event.TypeWithdrawn {
		h.logging.Named(Name).Debug(ctx, skippedMessage,
			logging.String(logging.MessageIDKey, m.ID),
			logging.String(logging.EventTypeKey, m.Attributes[workerbd.AttrEventType]),
		)
		return nil
	}

	withdrawn, err := event.ParseWithdrawn(m.Body)
	if err != nil {
		return xerrors.Wrap(apperror.ErrPermanent, err.Error())
	}

	if _, err := h.archive.ArchiveWithdrawal(ctx, user.ArchiveWithdrawalParams{
		UserID:  withdrawn.UserID,
		Payload: m.Body,
	}); err != nil {
		// 入力が不正なメッセージは、何度配送されても同じ結果になるため退避側へ回す。
		if xerrors.Is(err, apperror.ErrValidation) {
			return xerrors.Wrap(apperror.ErrPermanent, err.Error())
		}
		return err
	}

	return nil
}
