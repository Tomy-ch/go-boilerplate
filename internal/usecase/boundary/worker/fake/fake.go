// Package fake は、worker seam（Consumer / FailureHandler）の in-memory テストダブルを提供します。
// 実 broker 無しで engine の不変条件テストを green にするための 2nd impl です（SDK 非依存）。
package fake

import (
	"context"
	"sync"
	"time"

	"go-boilerplate/internal/usecase/boundary/worker"
)

// 実装漏れをコンパイル時に検出します。
var (
	_ worker.Consumer       = (*Fake)(nil)
	_ worker.FailureHandler = (*Fake)(nil)
)

// FailedRecord は、FailureHandler.Fail の呼び出し記録です（テスト検証用）。
type FailedRecord struct {
	Message worker.Message
	Cause   error
}

// Fake は、worker.Consumer と worker.FailureHandler を実装する in-memory テストダブルです。
// メッセージの投入・再配送・失敗注入と、Ack/Nack/Extend/Fail の呼び出し記録を提供します。
type Fake struct {
	mu sync.Mutex

	queue    []worker.Message          // 受信待ちのメッセージ
	inflight map[string]worker.Message // 受信済み・Ack/Nack 待ち（ID キー）
	delivery map[string]int            // ID ごとの配送回数（再配送で増加）

	acked     []string
	nacked    []string
	extends   map[string]int // ID ごとの Extend 呼び出し回数
	extendErr error          // 設定時、Extend が常にこのエラーを返す（H2 テスト用）
	failed    []FailedRecord

	receiveErrs []error       // 注入された Receive エラー（先頭から消費）
	notify      chan struct{} // long-poll 起床用のブロードキャスト
}

// New は、空の Fake を生成します。
func New() *Fake {
	return &Fake{
		inflight: make(map[string]worker.Message),
		delivery: make(map[string]int),
		extends:  make(map[string]int),
		notify:   make(chan struct{}),
	}
}

// Receive は、最大 max 件を取得します。キューが空の場合は投入 or ctx 完了までブロックします。
func (f *Fake) Receive(ctx context.Context, limit int) ([]worker.Message, error) {
	for {
		f.mu.Lock()
		if len(f.receiveErrs) > 0 {
			err := f.receiveErrs[0]
			f.receiveErrs = f.receiveErrs[1:]
			f.mu.Unlock()
			return nil, err
		}
		if len(f.queue) > 0 {
			n := min(limit, len(f.queue))
			out := make([]worker.Message, 0, n)
			for range n {
				m := f.queue[0]
				f.queue = f.queue[1:]
				f.delivery[m.ID]++
				m.ReceiveCount = f.delivery[m.ID]
				f.inflight[m.ID] = m
				out = append(out, m)
			}
			f.mu.Unlock()
			return out, nil
		}
		notify := f.notify
		f.mu.Unlock()

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-notify:
			// メッセージ投入 or 再配送で起床。ループ先頭で再評価する。
		}
	}
}

// Ack は、メッセージを in-flight から除去し、記録します。
func (f *Fake) Ack(_ context.Context, m worker.Message) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.inflight, m.ID)
	f.acked = append(f.acked, m.ID)
	return nil
}

// Nack は、メッセージを再配送（キュー末尾へ戻す）し、記録します。
func (f *Fake) Nack(_ context.Context, m worker.Message) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.inflight, m.ID)
	f.nacked = append(f.nacked, m.ID)
	f.queue = append(f.queue, m)
	f.signal()
	return nil
}

// Extend は、Extend の呼び出し回数を記録します（可視性の実時間延長は模さない）。
// SetExtendErr でエラーが設定されている場合はそのエラーを返します。
func (f *Fake) Extend(_ context.Context, m worker.Message, _ time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.extends[m.ID]++
	return f.extendErr
}

// Fail は、FailureHandler としての退避を記録します。
func (f *Fake) Fail(_ context.Context, m worker.Message, cause error) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failed = append(f.failed, FailedRecord{Message: m, Cause: cause})
	return nil
}

// --- テスト操作・検証用ヘルパー ---

// Enqueue は、メッセージをキューに投入します。
func (f *Fake) Enqueue(msgs ...worker.Message) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.queue = append(f.queue, msgs...)
	f.signal()
}

// FailReceiveOnce は、次回以降の Receive で返すエラーを 1 件キューイングします（複数回呼べば順に消費）。
func (f *Fake) FailReceiveOnce(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.receiveErrs = append(f.receiveErrs, err)
	f.signal()
}

// SetExtendErr は、以降の Extend が常に返すエラーを設定します（H2 の Extend 失敗テスト用）。
func (f *Fake) SetExtendErr(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.extendErr = err
}

// AckedIDs は、Ack されたメッセージ ID の一覧（呼び出し順）を返します。
func (f *Fake) AckedIDs() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.acked...)
}

// NackedIDs は、Nack されたメッセージ ID の一覧（呼び出し順）を返します。
func (f *Fake) NackedIDs() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.nacked...)
}

// ExtendCount は、指定 ID の Extend 呼び出し回数を返します。
func (f *Fake) ExtendCount(id string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.extends[id]
}

// Failed は、Fail の呼び出し記録（呼び出し順）を返します。
func (f *Fake) Failed() []FailedRecord {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]FailedRecord(nil), f.failed...)
}

// QueueLen は、受信待ちのメッセージ数を返します。
func (f *Fake) QueueLen() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.queue)
}

// InflightLen は、受信済み・未 Ack/Nack のメッセージ数を返します。
func (f *Fake) InflightLen() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.inflight)
}

// signal は、long-poll 中の Receive を起床させます。呼び出しは mu ロック下で行います。
func (f *Fake) signal() {
	close(f.notify)
	f.notify = make(chan struct{})
}
