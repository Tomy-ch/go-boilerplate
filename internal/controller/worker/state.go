package worker

import (
	"sync"

	"go-boilerplate/internal/usecase/boundary/worker"
)

type state struct {
	mu sync.Mutex

	name string
	args []string
}

// NewState は、新しい State インスタンスを生成します。
func NewState() worker.State {
	return &state{}
}

// Set は、起動対象の worker 名と引数を設定します。
func (s *state) Set(name string, args []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.name = name
	s.args = args
}

// Snapshot は、現在の起動対象をスナップショットとして取得します。
func (s *state) Snapshot() (string, []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.name, s.args
}
