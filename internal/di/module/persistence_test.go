package module

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	idempotencybndry "go-boilerplate/internal/usecase/boundary/idempotency"
	outboxbndry "go-boilerplate/internal/usecase/boundary/outbox"
	"go-boilerplate/internal/usecase/healthcheck/query"
)

func Test_persistenceModule_GraphIsValid(t *testing.T) {
	t.Parallel()

	// 永続化系（repository / query_service / system_cqrs）はリポジトリ追加で増える領域。
	// SQL や実 DB 挙動は infra 層のテストに任せ、ここでは各コンストラクタが DB ハンドル等の
	// 依存と正しく結線されることを確認する。
	opts := append(commonDeps(), persistenceModule())
	validateGraph(t, opts...)
}

func Test_persistenceModule(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("system_cqrs のヘルスチェック / 冪等 / outbox ストアを提供する", func(t *testing.T) {
			t.Parallel()

			var (
				dbsq        query.DBSystemCqrs
				idempotency idempotencybndry.Store
				outbox      outboxbndry.Store
			)

			validateGraph(t, append(commonDeps(), persistenceModule(),
				fx.Populate(&dbsq, &idempotency, &outbox))...)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未配線では outbox ストアが解決できずグラフ検証に失敗する", func(t *testing.T) {
			t.Parallel()

			var outbox outboxbndry.Store

			opts := append(commonDeps(), fx.Populate(&outbox), fx.NopLogger)
			require.Error(t, fx.ValidateApp(opts...))
		})
	})
}
