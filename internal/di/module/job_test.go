package module

import (
	"testing"

	"go-boilerplate/internal/di/shutdowner"

	"github.com/stretchr/testify/assert"
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
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}
