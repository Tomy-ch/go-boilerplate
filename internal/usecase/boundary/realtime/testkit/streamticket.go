package testkit

import (
	"context"
	"sync"
	"time"

	rt "go-boilerplate/internal/usecase/boundary/realtime"
)

var _ rt.StreamTicketStore = (*StreamTicketStore)(nil)

// StreamTicketStore は、in-memory の rt.StreamTicketStore です。読み取りは常に最新の書き込みを反映します。
type StreamTicketStore struct {
	mu      sync.Mutex
	tickets map[rt.TicketHash]rt.StreamTicket
}

// NewStreamTicketStore は、空の StreamTicketStore を生成します。
func NewStreamTicketStore() *StreamTicketStore {
	return &StreamTicketStore{tickets: map[rt.TicketHash]rt.StreamTicket{}}
}

// Save は、ticket を保存します。同じ Hash への再保存は上書きです。
func (s *StreamTicketStore) Save(_ context.Context, ticket rt.StreamTicket) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tickets[ticket.Hash] = ticket

	return nil
}

// Find は、hash に対応する ticket を返します。無い、または asOf 時点で ExpiresAt に達しているものは
// ok=false を返します。
func (s *StreamTicketStore) Find(_ context.Context, hash rt.TicketHash, asOf time.Time) (rt.StreamTicket, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ticket, ok := s.tickets[hash]
	if !ok || !asOf.Before(ticket.ExpiresAt) {
		return rt.StreamTicket{}, false, nil
	}

	return ticket, true, nil
}

// Invalidate は、subject × destination に発行された ticket をすべて落とします。該当が無くてもエラーになりません。
func (s *StreamTicketStore) Invalidate(_ context.Context, subject string, destination rt.StreamID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for hash, ticket := range s.tickets {
		if ticket.Subject == subject && ticket.Destination == destination {
			delete(s.tickets, hash)
		}
	}

	return nil
}

// Len は、保持している ticket の件数を返します。
func (s *StreamTicketStore) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return len(s.tickets)
}
