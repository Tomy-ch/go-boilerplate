package worker

import (
	"sync"

	"go-boilerplate/internal/usecase/boundary/worker"
)

type state struct {
	mu sync.Mutex

	name string
	args []string
	done chan error
}

// NewState は、新しい State インスタンスを生成します。
func NewState() worker.State {
	return &state{}
}

// Set は、起動対象の worker 名・引数と done チャネルを設定します。
func (s *state) Set(name string, args []string, done chan error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.name = name
	s.args = args
	s.done = done
}

// Snapshot は、現在の起動対象と done チャネルをスナップショットとして取得します。
func (s *state) Snapshot() (string, []string, chan error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.name, s.args, s.done
}
