// Package job は、ジョブ実行に関するコントローラを提供します。
package job

import (
	"context"
	"fmt"
	"sort"

	"go-boilerplate/internal/usecase/boundary/job"
	"go-boilerplate/pkg/xerrors"
)

var (
	// ErrDuplicateJob は、同名のジョブが重複登録された場合のエラーです。
	ErrDuplicateJob = xerrors.New("duplicate job")
	// ErrUnknownJob は、未登録のジョブ名が指定された場合のエラーです。
	ErrUnknownJob = xerrors.New("unknown job")
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
			return nil, xerrors.Wrap(ErrDuplicateJob, name)
		}
		m[name] = j
	}
	return &runner{registry: m}, nil
}

// Run は、指定されたジョブを実行します。
func (r *runner) Run(ctx context.Context, jobName string, args []string) error {
	j, ok := r.registry[jobName]
	if !ok {
		return xerrors.Wrap(ErrUnknownJob, fmt.Sprintf("%s (available: %v)", jobName, r.Names()))
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
