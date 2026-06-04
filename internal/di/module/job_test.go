package module

import (
	"context"
	"testing"

	"go-boilerplate/internal/di/shutdowner"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

func TestJobModule_GraphIsValid(t *testing.T) {
	t.Parallel()

	// ジョブ層はジョブ追加で増える領域。個々のジョブの振る舞いは controller/job 層のテストに任せ、
	// ここではジョブ群・ランナー・フックが依存と正しく結線されることを確認する。
	opts := append(commonDeps(),
		shutdowner.Module(),
		InfrastructureModule(), UsecaseModule(),
		JobModule(),
	)
	validateGraph(t, opts...)
}

func Test_provideJobs_AnnotatesIntoJobsGroup(t *testing.T) {
	t.Parallel()

	// provideJobs は渡したコンストラクタ群を group:"jobs" として登録する。
	// 登録した数だけグループに集約されることを確認する。
	type fakeJob struct{ name string }
	var jobs []*fakeJob

	app := fx.New(
		provideJobs(
			func() *fakeJob { return &fakeJob{name: "a"} },
			func() *fakeJob { return &fakeJob{name: "b"} },
		),
		fx.Populate(fx.Annotate(&jobs, fx.ParamTags(`group:"jobs"`))),
		fx.NopLogger,
	)

	require.NoError(t, app.Start(context.Background()))
	defer func() { require.NoError(t, app.Stop(context.Background())) }()

	assert.Len(t, jobs, 2)
}
