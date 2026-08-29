package hook

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/internal/di/lifecycle"
	"go-boilerplate/internal/logging"
	"go-boilerplate/pkg/xerrors"
)

var (
	errParticipant = xerrors.New("participant failed")
	errTeardown    = xerrors.New("teardown failed")
	errRunnerStop  = xerrors.New("runner stop failed")
)

// callLog は、参加者と HTTP の呼び出し順を記録します。
type callLog struct {
	mu    sync.Mutex
	calls []string
}

func (c *callLog) add(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.calls = append(c.calls, name)
}

func (c *callLog) get() []string {
	c.mu.Lock()
	defer c.mu.Unlock()

	return append([]string(nil), c.calls...)
}

// fixture は、記録する参加者と HTTP の start / stop を持つ serveLifecycle を組み立てます。
type fixture struct {
	lc  *serveLifecycle
	log *callLog
	// logCount は、msg のログの件数を返します。
	logCount func(msg string) int
}

func newFixture(t *testing.T) *fixture {
	t.Helper()

	logger, logs := logging.NewObservedTestLogger(t)
	calls := &callLog{}

	return &fixture{
		lc: &serveLifecycle{
			log:       logger,
			httpStart: func(context.Context) error { calls.add("http.start"); return nil },
			httpStop:  func(context.Context) error { calls.add("http.stop"); return nil },
		},
		log:      calls,
		logCount: func(msg string) int { return logs.FilterMessage(msg).Len() },
	}
}

func (f *fixture) probe(name string, err error) StartupProbe {
	return StartupProbe{Name: name, Probe: func(context.Context) error { f.log.add("probe." + name); return err }}
}

func (f *fixture) provisioner(name string, provisionErr, teardownErr error) Provisioner {
	return Provisioner{
		Name:      name,
		Provision: func(context.Context) error { f.log.add("provision." + name); return provisionErr },
		Teardown:  func(context.Context) error { f.log.add("teardown." + name); return teardownErr },
	}
}

func (f *fixture) runner(name string) boundRunner {
	start, stop := lifecycle.SupervisedRunner{
		OnStartAux: func() { f.log.add("runner.start." + name) },
		Body:       func(ctx context.Context) { <-ctx.Done() },
		OnStopAux:  func(context.Context) { f.log.add("runner.stop." + name) },
	}.Bind()

	return boundRunner{name: name, start: start, stop: stop}
}

func (f *fixture) drainer(name string, err error) Drainer {
	return Drainer{Name: name, Drain: func(context.Context) error { f.log.add("drain." + name); return err }}
}

func Test_newServeLifecycle(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Runner は Bind 済みの start / stop になり、参加者は渡した順に並ぶ", func(t *testing.T) {
			t.Parallel()

			log, _ := logging.NewObservedTestLogger(t)
			in := HTTPServerHooksIn{
				Log:     log,
				Runners: []Runner{{Name: "a"}, {Name: "b"}},
				Startup: []StartupProbe{
					{Name: "p", Probe: func(context.Context) error { return nil }},
				},
			}
			// newStartServerFunc / newStopServerFunc は nil の *http.Server でもクロージャを作るだけなので、ここでは呼ばない。
			lc := newServeLifecycle(in)

			require.Len(t, lc.runners, 2)
			assert.Equal(t, "a", lc.runners[0].name)
			assert.NotNil(t, lc.runners[0].start)
			assert.NotNil(t, lc.runners[1].stop)
			assert.Len(t, lc.startup, 1)
			assert.NotNil(t, lc.httpStart)
			assert.NotNil(t, lc.httpStop)
		})
	})
}

func Test_serveLifecycle_start(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("到達確認 → resource 作成 → 常駐処理 → HTTP listen の順に起動する", func(t *testing.T) {
			t.Parallel()

			f := newFixture(t)
			f.lc.startup = []StartupProbe{f.probe("store", nil)}
			f.lc.provisioners = []Provisioner{f.provisioner("lease", nil, nil), f.provisioner("inbox", nil, nil)}
			f.lc.runners = []boundRunner{f.runner("consumer")}

			require.NoError(t, f.lc.start(t.Context()))
			assert.Equal(
				t,
				[]string{"probe.store", "provision.lease", "provision.inbox", "runner.start.consumer", "http.start"},
				f.log.get(),
			)
		})

		t.Run("参加者がゼロなら HTTP listen だけ", func(t *testing.T) {
			t.Parallel()

			f := newFixture(t)
			require.NoError(t, f.lc.start(t.Context()))
			assert.Equal(t, []string{"http.start"}, f.log.get())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("到達確認の失敗は何も作らずに失敗する", func(t *testing.T) {
			t.Parallel()

			f := newFixture(t)
			f.lc.startup = []StartupProbe{f.probe("store", errParticipant)}
			f.lc.provisioners = []Provisioner{f.provisioner("inbox", nil, nil)}

			err := f.lc.start(t.Context())
			require.ErrorIs(t, err, errParticipant)
			assert.Contains(t, err.Error(), "startup probe store")
			assert.Equal(t, []string{"probe.store"}, f.log.get())
		})

		t.Run("resource 作成の失敗は作成済みの参加者を逆順に片付けてから失敗する", func(t *testing.T) {
			t.Parallel()

			f := newFixture(t)
			f.lc.provisioners = []Provisioner{
				f.provisioner(
					"lease",
					nil,
					nil,
				), f.provisioner("inbox", errParticipant, nil), f.provisioner("never", nil, nil),
			}

			err := f.lc.start(t.Context())
			require.ErrorIs(t, err, errParticipant)
			assert.Equal(t, []string{"provision.lease", "provision.inbox", "teardown.lease"}, f.log.get())
		})

		t.Run("HTTP listen の失敗は常駐処理を止め resource を片付けてから失敗する", func(t *testing.T) {
			t.Parallel()

			f := newFixture(t)
			f.lc.httpStart = func(context.Context) error { f.log.add("http.start"); return errParticipant }
			f.lc.provisioners = []Provisioner{f.provisioner("lease", nil, nil), f.provisioner("inbox", nil, nil)}
			f.lc.runners = []boundRunner{f.runner("heartbeat"), f.runner("consumer")}

			err := f.lc.start(t.Context())
			require.ErrorIs(t, err, errParticipant)
			assert.Equal(t, []string{
				"provision.lease", "provision.inbox", "runner.start.heartbeat", "runner.start.consumer", "http.start",
				"runner.stop.consumer", "runner.stop.heartbeat", "teardown.inbox", "teardown.lease",
			}, f.log.get())
		})
	})
}

func Test_serveLifecycle_stop(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("drain → 常駐処理の停止 → resource の片付け → HTTP shutdown の順で、drain が終わるまで shutdown は始まらない", func(t *testing.T) {
			t.Parallel()

			f := newFixture(t)
			drained := false
			f.lc.drainers = []Drainer{{Name: "sse", Drain: func(context.Context) error {
				f.log.add("drain.sse")
				drained = true

				return nil
			}}}
			f.lc.httpStop = func(context.Context) error {
				assert.True(t, drained, "HTTP shutdown は drain の完了後")
				f.log.add("http.stop")

				return nil
			}
			f.lc.provisioners = []Provisioner{f.provisioner("lease", nil, nil), f.provisioner("inbox", nil, nil)}
			f.lc.runners = []boundRunner{f.runner("heartbeat"), f.runner("consumer")}
			require.NoError(t, f.lc.start(t.Context()))
			f.log.calls = nil

			require.NoError(t, f.lc.stop(t.Context()))
			assert.Equal(t, []string{
				"drain.sse", "runner.stop.consumer", "runner.stop.heartbeat", "teardown.inbox", "teardown.lease", "http.stop",
			}, f.log.get())
		})

		t.Run("参加者がゼロなら HTTP shutdown だけ", func(t *testing.T) {
			t.Parallel()

			f := newFixture(t)
			require.NoError(t, f.lc.stop(t.Context()))
			assert.Equal(t, []string{"http.stop"}, f.log.get())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("drain と片付けの失敗は記録して続け、まとめて返す。HTTP shutdown は必ず呼ぶ", func(t *testing.T) {
			t.Parallel()

			f := newFixture(t)
			f.lc.drainers = []Drainer{f.drainer("sse", errParticipant)}
			f.lc.provisioners = []Provisioner{f.provisioner("inbox", nil, errTeardown)}
			f.lc.runners = []boundRunner{{name: "bad", start: func(context.Context) error { return nil }, stop: func(context.Context) error {
				f.log.add("runner.stop.bad")

				return errRunnerStop
			}}}

			err := f.lc.stop(t.Context())
			require.ErrorIs(t, err, errParticipant)
			require.ErrorIs(t, err, errRunnerStop, "常駐処理の停止失敗も返す")
			require.ErrorIs(t, err, errTeardown, "片付けの失敗も返す")
			assert.Equal(t, []string{"drain.sse", "runner.stop.bad", "teardown.inbox", "http.stop"}, f.log.get())
			assert.Equal(t, 1, f.logCount("serve drainer failed"))
			assert.Equal(t, 1, f.logCount("serve provisioner failed to tear down"))
		})

		t.Run("HTTP shutdown の失敗は返す", func(t *testing.T) {
			t.Parallel()

			f := newFixture(t)
			f.lc.httpStop = func(context.Context) error { return errParticipant }

			require.ErrorIs(t, f.lc.stop(t.Context()), errParticipant)
		})
	})
}

func Test_serveLifecycle_stopRunners(t *testing.T) {
	t.Parallel()

	t.Run("先頭 n 個を逆順に止め、失敗は記録する", func(t *testing.T) {
		t.Parallel()

		f := newFixture(t)
		failing := boundRunner{
			name:  "bad",
			start: func(context.Context) error { return nil },
			stop: func(context.Context) error {
				f.log.add("runner.stop.bad")

				return errParticipant
			},
		}
		f.lc.runners = []boundRunner{f.runner("a"), failing, f.runner("never")}
		require.NoError(t, f.lc.runners[0].start(t.Context()))

		errs := f.lc.stopRunners(t.Context(), 2)

		require.Len(t, errs, 1)
		assert.Equal(t, []string{"runner.start.a", "runner.stop.bad", "runner.stop.a"}, f.log.get())
		assert.Equal(t, 1, f.logCount("serve runner failed to stop"))
	})
}

func Test_serveLifecycle_teardown(t *testing.T) {
	t.Parallel()

	t.Run("先頭 n 個を逆順に片付け、失敗は記録する", func(t *testing.T) {
		t.Parallel()

		f := newFixture(t)
		f.lc.provisioners = []Provisioner{
			f.provisioner("a", nil, errParticipant),
			f.provisioner("b", nil, nil),
			f.provisioner("never", nil, nil),
		}

		errs := f.lc.teardown(t.Context(), 2)

		require.Len(t, errs, 1)
		assert.Equal(t, []string{"teardown.b", "teardown.a"}, f.log.get())
		assert.Equal(t, 1, f.logCount("serve provisioner failed to tear down"))
	})
}
