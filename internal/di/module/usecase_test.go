package module

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"go-boilerplate/internal/usecase/healthcheck"
	"go-boilerplate/internal/usecase/idempotency"
	"go-boilerplate/internal/usecase/outbox"
)

func TestUsecaseModule_GraphIsValid(t *testing.T) {
	t.Parallel()

	// 各ユースケースの振る舞いは usecase 層のテストに任せ、
	// ここではコンストラクタがリポジトリ等の依存と正しく結線されることを確認する。
	opts := append(commonDeps(), InfrastructureModule(), UsecaseModule())
	validateGraph(t, opts...)
}

func TestUsecaseModule(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("全プロファイル共通のヘルスチェック / 冪等 / outbox ユースケースを提供する", func(t *testing.T) {
			t.Parallel()

			var (
				health healthcheck.Usecase
				gc     idempotency.GCUsecase
				emit   outbox.EmitUsecase
				replay outbox.ReplayUsecase
			)

			validateGraph(t, append(commonDeps(), InfrastructureModule(), UsecaseModule(),
				fx.Populate(&health, &gc, &emit, &replay))...)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未配線では outbox の emit ユースケースが解決できずグラフ検証に失敗する", func(t *testing.T) {
			t.Parallel()

			var emit outbox.EmitUsecase

			opts := append(commonDeps(), InfrastructureModule(), fx.Populate(&emit), fx.NopLogger)
			require.Error(t, fx.ValidateApp(opts...))
		})
	})
}
