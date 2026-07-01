package worker

import (
	"context"
	"sync"

	"go-boilerplate/internal/usecase/boundary/worker"
)

// keyedDispatcher は、PartitionKey ごとにメッセージを直列化しつつ全体並列で処理します（B3）。
//   - PartitionKey が空: そのまま並列に proc を起動。
//   - PartitionKey が非空: key ごとの単一 goroutine が FIFO で proc を順次実行。
//
// in-flight トークンは呼び出し側（poll loop）が dispatch 前に取得済みで、proc が解放します。
type keyedDispatcher struct {
	mu      sync.Mutex
	runners map[string]*keyRunner
	buffer  int
	proc    func(ctx context.Context, m worker.Message)
}

type keyRunner struct {
	ch      chan worker.Message
	pending int
}

// newKeyedDispatcher は、keyedDispatcher を生成します。buffer は key ごとのキュー長です。
func newKeyedDispatcher(buffer int, proc func(ctx context.Context, m worker.Message)) *keyedDispatcher {
	return &keyedDispatcher{
		runners: make(map[string]*keyRunner),
		buffer:  buffer,
		proc:    proc,
	}
}

// dispatch は、メッセージを並列 or key 直列で処理へ回します。
func (kd *keyedDispatcher) dispatch(ctx context.Context, m worker.Message) {
	if m.PartitionKey == "" {
		go kd.proc(ctx, m)
		return
	}

	kd.mu.Lock()
	r, ok := kd.runners[m.PartitionKey]
	if !ok {
		r = &keyRunner{ch: make(chan worker.Message, kd.buffer)}
		kd.runners[m.PartitionKey] = r
		go kd.runKey(ctx, m.PartitionKey, r)
	}
	r.pending++
	kd.mu.Unlock()

	r.ch <- m
}

// runKey は、特定 key のメッセージを FIFO で順次処理し、滞留が無くなれば自身を破棄します。
func (kd *keyedDispatcher) runKey(ctx context.Context, key string, r *keyRunner) {
	for {
		m := <-r.ch
		kd.proc(ctx, m)

		kd.mu.Lock()
		r.pending--
		if r.pending == 0 {
			delete(kd.runners, key)
			kd.mu.Unlock()
			return
		}
		kd.mu.Unlock()
	}
}
