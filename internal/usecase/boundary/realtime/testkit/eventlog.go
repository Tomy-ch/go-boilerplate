// Package testkit は、Realtime Delivery の境界をテストから駆動するための実装を提供します。
// 本番の配線には現れません。
package testkit

import (
	"context"
	"sort"
	"sync"

	"go-boilerplate/internal/apperror"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	"go-boilerplate/pkg/xerrors"
)

// defaultReadLimit は、ReadAfterQuery.Limit が 0 以下のときに返す件数の上限です。
const defaultReadLimit = 32

var (
	_ rt.EventLogStore = (*EventLog)(nil)

	// errEventLogUnavailable は、SetUnavailable(true) の間に読み書きが返すエラーです。
	errEventLogUnavailable = xerrors.Wrap(apperror.ErrUnavailable, "testkit: event log is unavailable")
)

// EventLog は、in-memory の rt.EventLogStore です。読み取りは常に最新の書き込みを反映します
// （実装は強い一貫性の読み取りとして振る舞います）。
//
// 正規の Append では作れない状態を組み立てる専用の口（Seed / SetUnavailable / Hold /
// SeedAppendedThrough）を持ちます。
// 各口の使い分けは README を参照してください。
type EventLog struct {
	mu          sync.Mutex
	streams     map[rt.StreamID][]rt.DeliveryEvent
	appended    map[rt.StreamID]rt.Sequence
	unavailable bool
	// held は、閉じられるまで読み取りを待たせる関門です。nil のときは待たせません。
	held chan struct{}
}

// NewEventLog は、空の EventLog を生成します。
func NewEventLog() *EventLog {
	return &EventLog{
		streams:  map[rt.StreamID][]rt.DeliveryEvent{},
		appended: map[rt.StreamID]rt.Sequence{},
	}
}

// Seed は、Validate も冪等判定も通さずに event を置きます。飛び番や保持期間外の位置など、
// 正規の Append では作れない状態をテストが直接組み立てるための口です。
func (l *EventLog) Seed(events ...rt.DeliveryEvent) {
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, e := range events {
		l.streams[e.StreamID] = insert(l.streams[e.StreamID], e)
	}
}

// Hold は、以降の読み取りを、返った関数が呼ばれるまで待たせます。読み取りが進行中である状態を
// テストが作るための口です。
func (l *EventLog) Hold() func() {
	l.mu.Lock()
	gate := make(chan struct{})
	l.held = gate
	l.mu.Unlock()

	var once sync.Once

	return func() {
		once.Do(func() {
			l.mu.Lock()
			if l.held == gate {
				l.held = nil
			}
			l.mu.Unlock()
			close(gate)
		})
	}
}

// SetUnavailable は、以降の読み書きが apperror.ErrUnavailable を返すかどうかを切り替えます。
func (l *EventLog) SetUnavailable(v bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.unavailable = v
}

// Append は、event を (StreamID, Sequence) へ 1 度だけ書きます。同じ位置に同じ EventID があれば成功、
// 異なる EventID があれば rt.ErrSequenceConflict を返します。
func (l *EventLog) Append(_ context.Context, event rt.DeliveryEvent) error {
	if err := event.Validate(); err != nil {
		return err
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.unavailable {
		return errEventLogUnavailable
	}

	for _, e := range l.streams[event.StreamID] {
		if e.Sequence == event.Sequence {
			if e.EventID == event.EventID {
				return nil
			}

			return rt.ErrSequenceConflict
		}
	}

	l.streams[event.StreamID] = insert(l.streams[event.StreamID], event)
	if event.Sequence > l.appended[event.StreamID] {
		l.appended[event.StreamID] = event.Sequence
	}

	return nil
}

// AppendedThrough は、Append で書いた最大の位置を返します。Seed は正規の追記ではないので数えません。
func (l *EventLog) AppendedThrough(_ context.Context, streamID rt.StreamID) (rt.Sequence, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.unavailable {
		return 0, errEventLogUnavailable
	}

	return l.appended[streamID], nil
}

// SeedAppendedThrough は、Append を経ずに追記済みの位置を置きます。保持期間で event が消えたあとの
// 状態など、正規の Append では作れない状態をテストが直接組み立てるための口です。
func (l *EventLog) SeedAppendedThrough(streamID rt.StreamID, seq rt.Sequence) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.appended[streamID] = seq
}

// ReadAfter は、q.After より後ろの event を sequence 昇順に最大 q.Limit 件返します。
func (l *EventLog) ReadAfter(ctx context.Context, q rt.ReadAfterQuery) (rt.ReadAfterResult, error) {
	if err := l.awaitRelease(ctx); err != nil {
		return rt.ReadAfterResult{}, err
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.unavailable {
		return rt.ReadAfterResult{}, errEventLogUnavailable
	}

	limit := int(q.Limit)
	if limit <= 0 {
		limit = defaultReadLimit
	}

	var found []rt.DeliveryEvent
	for _, e := range l.streams[q.StreamID] {
		if e.Sequence > q.After {
			found = append(found, e)
		}
	}

	if len(found) > limit {
		return rt.ReadAfterResult{Events: found[:limit], HasMore: true}, nil
	}

	return rt.ReadAfterResult{Events: found}, nil
}

// Latest は、stream の最後の event を返します。1 件も無ければ ok=false を返します。
func (l *EventLog) Latest(_ context.Context, streamID rt.StreamID) (rt.DeliveryEvent, bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.unavailable {
		return rt.DeliveryEvent{}, false, errEventLogUnavailable
	}

	events := l.streams[streamID]
	if len(events) == 0 {
		return rt.DeliveryEvent{}, false, nil
	}

	return events[len(events)-1], true, nil
}

// Find は、stream の指定 sequence の event を返します。無ければ ok=false を返します。
func (l *EventLog) Find(_ context.Context, streamID rt.StreamID, seq rt.Sequence) (rt.DeliveryEvent, bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.unavailable {
		return rt.DeliveryEvent{}, false, errEventLogUnavailable
	}

	for _, e := range l.streams[streamID] {
		if e.Sequence == seq {
			return e, true, nil
		}
	}

	return rt.DeliveryEvent{}, false, nil
}

// awaitRelease は、Hold が掛かっていれば解けるまで待ちます。
func (l *EventLog) awaitRelease(ctx context.Context) error {
	l.mu.Lock()
	gate := l.held
	l.mu.Unlock()

	if gate == nil {
		return nil
	}

	select {
	case <-gate:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// insert は、sequence 昇順を保ったまま event を差し込みます。同じ位置の event は置き換えます。
func insert(events []rt.DeliveryEvent, e rt.DeliveryEvent) []rt.DeliveryEvent {
	for i, cur := range events {
		if cur.Sequence == e.Sequence {
			events[i] = e

			return events
		}
	}

	events = append(events, e)
	sort.Slice(events, func(i, j int) bool { return events[i].Sequence < events[j].Sequence })

	return events
}
