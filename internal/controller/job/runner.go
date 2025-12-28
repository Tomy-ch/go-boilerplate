// Package job は、ジョブ実行に関するコントローラを提供します。
package job

import (
	"context"
	"fmt"
	"sort"

	"boilerplate-go/internal/usecase/support/job"
	"boilerplate-go/pkg/xerrors"
)

// runner は、ジョブの実行を管理する構造体です。
type runner struct {
	registry map[string]job.Job
}

// NewRunner は、Runnerを生成します。
func NewRunner(jobs []job.Job) (job.Runner, error) {
	m := make(map[string]job.Job, len(jobs))
	for _, j := range jobs {
		name := j.Name()
		if _, exists := m[name]; exists {
			return nil, xerrors.New("duplicate job: " + name)
		}
		m[name] = j
	}
	return &runner{registry: m}, nil
}

// Run は、指定されたジョブを実行します。
func (r *runner) Run(ctx context.Context, jobName string, args []string) error {
	j, ok := r.registry[jobName]
	if !ok {
		return xerrors.New(fmt.Sprintf("unknown job: %s (available: %v)", jobName, r.Names()))
	}
	return j.Execute(ctx, args)
}

// Names は、登録されているジョブの名前一覧を返します。
func (r *runner) Names() []string {
	out := make([]string, 0, len(r.registry))
	for k := range r.registry {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
