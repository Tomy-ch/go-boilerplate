// Package withdrawalarchive は、退会イベントを消費して退会証跡を保存する worker を提供します。
// outbox が publish した user.withdrawn.v1 を broker 経由で受け取る、consuming 端の実装例です。
package withdrawalarchive

import (
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	workerbd "go-boilerplate/internal/usecase/boundary/worker"
	"go-boilerplate/internal/usecase/user"
)

// Name は、この worker の名前です（`worker <name>` の引数で選択され、metric の worker ラベルにもなります）。
const Name = "withdrawal-archive"

var _ workerbd.Worker = (*workerImpl)(nil)

// workerImpl は、退会証跡 worker の seam 実装です。
type workerImpl struct {
	consumer workerbd.Consumer
	failure  workerbd.FailureHandler
	handler  workerbd.Handler
}

// New は、退会証跡 worker を初期化します。
// failure に nil を渡した場合、Permanent メッセージの退避は engine 既定の扱いになります。
func New(
	consumer workerbd.Consumer,
	failure workerbd.FailureHandler,
	archive user.ArchiveUsecase,
	tf observability.TracerFactory,
	logger logging.Logger,
) workerbd.Worker {
	return &workerImpl{
		consumer: consumer,
		failure:  failure,
		handler:  newHandler(archive, tf, logger),
	}
}

// Name は、この worker の名前を返します。
func (w *workerImpl) Name() string { return Name }

// Consumer は、この worker が消費するキューの Consumer を返します。
func (w *workerImpl) Consumer() workerbd.Consumer { return w.consumer }

// Handler は、業務処理を返します。
func (w *workerImpl) Handler() workerbd.Handler { return w.handler }

// FailureHandler は、Permanent メッセージの退避先を返します。
func (w *workerImpl) FailureHandler() workerbd.FailureHandler { return w.failure }
