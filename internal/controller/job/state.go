package job

import (
	"sync"

	"boilerplate-go/internal/usecase/interface/job"
)

type state struct {
	mu sync.Mutex

	name string
	args []string
	done chan error
}

// NewState は、新しい State インスタンスを生成します。
func NewState() job.State {
	return &state{}
}

// Set は、ジョブの状態を設定します。
func (s *state) Set(name string, args []string, done chan error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.name = name
	s.args = args
	s.done = done
}

// Snapshot は、現在のジョブの状態をスナップショットとして取得します。
func (s *state) Snapshot() (string, []string, chan error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.name, s.args, s.done
}
