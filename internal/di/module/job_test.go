package module

import (
	"testing"

	"go-boilerplate/internal/di/shutdowner"
	"go-boilerplate/internal/usecase/boundary/job"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

func TestJobModule_GraphIsValid(t *testing.T) {
	t.Parallel()

	// 個々のジョブの振る舞いは controller/job 層のテストに任せ、
	// ここではジョブ群・ランナー・フックが依存と正しく結線されることを確認する。
	opts := append(commonDeps(),
		shutdowner.Module(),
		InfrastructureModule(), UsecaseModule(),
		JobModule(),
	)
	validateGraph(t, opts...)
}

func Test_provideJobs(t *testing.T) {
	t.Parallel()

	type fakeJob struct{ name string }

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("渡した全コンストラクタのジョブが jobs グループへ集まる", func(t *testing.T) {
			t.Parallel()

			got := collectGroup[*fakeJob](t, `group:"jobs"`, provideJobs(
				func() *fakeJob { return &fakeJob{name: "a"} },
				func() *fakeJob { return &fakeJob{name: "b"} },
			))

			names := make([]string, 0, len(got))
			for _, j := range got {
				names = append(names, j.name)
			}
			assert.ElementsMatch(t, []string{"a", "b"}, names)
		})

		t.Run("コンストラクタが 0 個の場合は何も登録しない", func(t *testing.T) {
			t.Parallel()

			assert.Empty(t, collectGroup[*fakeJob](t, `group:"jobs"`, provideJobs()))
		})
	})
}

func TestJobModule(t *testing.T) {
	t.Parallel()

	jobDeps := func() []fx.Option {
		return append(commonDeps(), shutdowner.Module(), InfrastructureModule(), UsecaseModule())
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ジョブ実行に必要な Runner と State を提供する", func(t *testing.T) {
			t.Parallel()

			var (
				runner job.Runner
				state  job.State
			)

			validateGraph(t, append(jobDeps(), JobModule(), fx.Populate(&runner, &state))...)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未配線では Runner が解決できずグラフ検証に失敗する", func(t *testing.T) {
			t.Parallel()

			var runner job.Runner

			opts := append(jobDeps(), fx.Populate(&runner), fx.NopLogger)
			require.Error(t, fx.ValidateApp(opts...))
		})
	})
}
