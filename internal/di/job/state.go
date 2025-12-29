package job

import "sync"

type State struct {
	mu sync.Mutex

	name string
	args []string
	done chan error
}

// NewState は、新しい State インスタンスを生成します。
func NewState() *State {
	return &State{}
}

// Set は、ジョブの状態を設定します。
func (s *State) Set(name string, args []string, done chan error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.name = name
	s.args = args
	s.done = done
}

// Snapshot は、現在のジョブの状態をスナップショットとして取得します。
func (s *State) Snapshot() (string, []string, chan error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.name, s.args, s.done
}
