package testkit

import (
	"context"
	"sync"
	"time"

	rt "go-boilerplate/internal/usecase/boundary/realtime"
)

var _ rt.StreamTicketStore = (*StreamTicketStore)(nil)

// StreamTicketStore は、in-memory の rt.StreamTicketStore です。読み取りは常に最新の書き込みを反映します。
//
// 失効の検証に要るのは「Invalidate した subject × destination の ticket が Find から消える」ことだけなので、
// 期限切れの掃除は持たず、判定は Find の asOf で行います（本物の store と同じ切り分け）。
type StreamTicketStore struct {
	mu      sync.Mutex
	tickets map[rt.TicketHash]rt.StreamTicket
}

// NewStreamTicketStore は、空の StreamTicketStore を生成します。
func NewStreamTicketStore() *StreamTicketStore {
	return &StreamTicketStore{tickets: map[rt.TicketHash]rt.StreamTicket{}}
}

func (s *StreamTicketStore) Save(_ context.Context, ticket rt.StreamTicket) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tickets[ticket.Hash] = ticket

	return nil
}

func (s *StreamTicketStore) Find(_ context.Context, hash rt.TicketHash, asOf time.Time) (rt.StreamTicket, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ticket, ok := s.tickets[hash]
	if !ok || !asOf.Before(ticket.ExpiresAt) {
		return rt.StreamTicket{}, false, nil
	}

	return ticket, true, nil
}

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

// Len は、保持している ticket の件数を返します。失効が「消した」ことを件数でも確かめるための口です。
func (s *StreamTicketStore) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return len(s.tickets)
}
