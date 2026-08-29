package hook

import (
	"context"
	"net/http"

	"go.uber.org/fx"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/di/lifecycle"
	"go-boilerplate/internal/di/server/extension"
	"go-boilerplate/internal/logging"
	"go-boilerplate/pkg/xerrors"
)

const serveLifecycleLoggerName = "server.Lifecycle"

// HTTPServerHooksIn は、serve instance のライフサイクル hook が受け取る依存です。
// 参加者はいずれも任意（値が無くてもよい value group）で、1 つも無ければ HTTP サーバーだけを起動・停止します。
type HTTPServerHooksIn struct {
	fx.In

	Srv    *http.Server
	Reg    lifecycle.Registrar
	Log    logging.Logger
	AppCfg *config.ApplicationConfig
	SecCfg *config.SecurityConfig
	SrvCfg *config.ServerConfig
	OSCfg  *config.OperatingSystemConfig
	// Applied は、サーバー機能の拡張が適用されたことを示すトークンです。
	Applied *extension.AppliedServerExtends

	Startup      []StartupProbe `group:"serve.startup"`
	Provisioners []Provisioner  `group:"serve.provisioners"`
	Runners      []Runner       `group:"serve.runners"`
	Drainers     []Drainer      `group:"serve.drainers"`
}

// boundRunner は、Bind 済みの常駐処理です。
type boundRunner struct {
	name  string
	start func(ctx context.Context) error
	stop  func(ctx context.Context) error
}

// serveLifecycle は、参加者と HTTP サーバーの起動・停止を順序どおりに実行します。
type serveLifecycle struct {
	log          logging.Logger
	startup      []StartupProbe
	provisioners []Provisioner
	runners      []boundRunner
	drainers     []Drainer
	httpStart    func(ctx context.Context) error
	httpStop     func(ctx context.Context) error
}

// RegisterHTTPServerHooks は、serve instance の起動・停止の順序を 1 箇所で決めて登録します。
//
//	Start: 依存の到達確認 → 受信 resource の作成 → 常駐処理の開始 → HTTP listen
//	Stop:  drain（新規接続の拒否を含む）→ 常駐処理の停止 → 受信 resource の片付け → HTTP shutdown
//
// fx の OnStop は登録の逆順で走るため、順序を hook の登録順に委ねず、start / stop 各 1 本の中で明示的に
// 呼びます（参加者が増減しても順序が変わらず、drain が HTTP shutdown より前に完了することを固定する）。
func RegisterHTTPServerHooks(in HTTPServerHooksIn) {
	lc := newServeLifecycle(in)
	in.Reg.RegisterStart(lc.start)
	in.Reg.RegisterStop(lc.stop)
}

func newServeLifecycle(in HTTPServerHooksIn) *serveLifecycle {
	runners := make([]boundRunner, 0, len(in.Runners))
	for _, r := range in.Runners {
		start, stop := r.Runner.Bind()
		runners = append(runners, boundRunner{name: r.Name, start: start, stop: stop})
	}

	return &serveLifecycle{
		log:          in.Log.Named(serveLifecycleLoggerName).CallerSkip(serverCallerSkip),
		startup:      in.Startup,
		provisioners: in.Provisioners,
		runners:      runners,
		drainers:     in.Drainers,
		httpStart:    newStartServerFunc(in.Srv, in.Log, in.AppCfg, in.SecCfg, in.SrvCfg, in.OSCfg),
		httpStop:     newStopServerFunc(in.Srv, in.Log, in.OSCfg),
	}
}

// start は、到達確認 → resource 作成 → 常駐処理 → HTTP listen の順に起動します。
// 途中で失敗したら、そこまでに起動・作成したものを逆順に止めて片付けてから失敗を返します。
func (l *serveLifecycle) start(ctx context.Context) error {
	for _, p := range l.startup {
		if err := p.Probe(ctx); err != nil {
			return xerrors.Wrap(err, "serve startup probe "+p.Name+" failed")
		}
	}

	provisioned := 0
	for _, p := range l.provisioners {
		if err := p.Provision(ctx); err != nil {
			l.teardown(ctx, provisioned)

			return xerrors.Wrap(err, "serve provisioner "+p.Name+" failed")
		}

		provisioned++
	}

	started := 0
	for _, r := range l.runners {
		if err := r.start(ctx); err != nil {
			l.stopRunners(ctx, started)
			l.teardown(ctx, provisioned)

			return xerrors.Wrap(err, "serve runner "+r.name+" failed to start")
		}

		started++
	}

	if err := l.httpStart(ctx); err != nil {
		l.stopRunners(ctx, started)
		l.teardown(ctx, provisioned)

		return err
	}

	return nil
}

// stop は、drain → 常駐処理の停止 → resource の片付け → HTTP shutdown の順に停止します。
// 参加者の失敗は記録して次へ進み（片付けを途中で止めると resource が残る）、HTTP shutdown まで終えてから
// すべての失敗をまとめて返します。
func (l *serveLifecycle) stop(ctx context.Context) error {
	var errs []error

	for _, d := range l.drainers {
		if err := d.Drain(ctx); err != nil {
			l.log.Error(
				ctx,
				"serve drainer failed",
				logging.String("participant", d.Name),
				logging.Error(logging.ErrorKey, err),
			)
			errs = append(errs, err)
		}
	}

	errs = append(errs, l.stopRunners(ctx, len(l.runners))...)
	errs = append(errs, l.teardown(ctx, len(l.provisioners))...)

	if err := l.httpStop(ctx); err != nil {
		errs = append(errs, err)
	}

	return xerrors.Join(errs...)
}

// stopRunners は、先頭から n 個の常駐処理を逆順に止め、失敗を記録して返します（止め切るまで途中で止めない）。
func (l *serveLifecycle) stopRunners(ctx context.Context, n int) []error {
	var errs []error

	for i := n - 1; i >= 0; i-- {
		if err := l.runners[i].stop(ctx); err != nil {
			l.log.Error(
				ctx,
				"serve runner failed to stop",
				logging.String("participant", l.runners[i].name),
				logging.Error(logging.ErrorKey, err),
			)
			errs = append(errs, err)
		}
	}

	return errs
}

// teardown は、先頭から n 個の参加者の resource を逆順に片付け、失敗を記録して返します（片付け切るまで途中で止めない）。
func (l *serveLifecycle) teardown(ctx context.Context, n int) []error {
	var errs []error

	for i := n - 1; i >= 0; i-- {
		p := l.provisioners[i]
		if err := p.Teardown(ctx); err != nil {
			l.log.Error(
				ctx,
				"serve provisioner failed to tear down",
				logging.String("participant", p.Name),
				logging.Error(logging.ErrorKey, err),
			)
			errs = append(errs, err)
		}
	}

	return errs
}
