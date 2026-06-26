package lifecycle

import (
	"context"
)

// SupervisedRunner は、detached goroutine で常駐する background runner を fx ライフサイクルへ
// 結線する共通プリミティブです。job / worker / outbox relay の各 hook が共有します。
//
//   - OnStart: [SupervisedRunner.OnStartAux]（任意）を同期実行した後、[SupervisedRunner.Body] を
//     goroutine で起動する（OnStart はブロックしない）。
//   - OnStop:  実行 context をキャンセルし、Body の完了を stopCtx（grace）の範囲で待ち、
//     その後 [SupervisedRunner.OnStopAux]（任意）を実行する。
//
// 実行 context は context.Background() から WithCancel で派生させます。これにより
//   - OnStart 完了後に fx が startCtx をキャンセルしても Body は巻き込まれず（detached）、
//   - OnStop の cancel でのみ Body の context が切れて中断できる（停止時キャンセル）。
type SupervisedRunner struct {
	// Body は、background ループ本体です。渡される context は OnStop でキャンセルされます。
	// nil の場合は何も起動しません。
	Body func(ctx context.Context)
	// OnStartAux は、Body 起動前に OnStart 内で同期実行される任意の処理です（例: health listener 起動）。
	OnStartAux func()
	// OnStopAux は、drain 完了後に実行される任意の処理です（例: health listener 停止）。
	// 引数の stopCtx は OnStop の grace context です。
	OnStopAux func(stopCtx context.Context)
}

// Register は、SupervisedRunner を reg のライフサイクルへ登録します。
func (s SupervisedRunner) Register(reg Registrar) {
	// 実行 context は OnStop でのみキャンセルする（OnStart 完了後の startCtx キャンセルに巻き込まれない）。
	runCtx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	reg.RegisterStart(func(_ context.Context) error {
		if s.OnStartAux != nil {
			s.OnStartAux()
		}
		go func() {
			defer close(done)
			if s.Body != nil {
				s.Body(runCtx)
			}
		}()
		return nil
	})

	reg.RegisterStop(func(stopCtx context.Context) error {
		cancel()
		select {
		case <-done: // Body 完了（drain 完了）
		case <-stopCtx.Done(): // 猶予切れ
		}
		if s.OnStopAux != nil {
			s.OnStopAux(stopCtx)
		}
		return nil
	})
}
