package module

import (
	"testing"

	"github.com/stretchr/testify/assert" // sample-api:line
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/mock/gomock" // sample-api:line

	"go-boilerplate/internal/config"                                                 // sample-api:line
	mock_category "go-boilerplate/internal/domain/product/category/mock"             // sample-api:line
	mock_product "go-boilerplate/internal/domain/product/mock"                       // sample-api:line
	mock_status "go-boilerplate/internal/domain/product/status/mock"                 // sample-api:line
	"go-boilerplate/internal/observability"                                          // sample-api:line
	mock_authz "go-boilerplate/internal/usecase/boundary/authz/mock"                 // sample-api:line
	mock_objectstorage "go-boilerplate/internal/usecase/boundary/objectstorage/mock" // sample-api:line
	mock_tx "go-boilerplate/internal/usecase/boundary/tx/mock"                       // sample-api:line
	"go-boilerplate/internal/usecase/healthcheck"
	"go-boilerplate/internal/usecase/idempotency"
	"go-boilerplate/internal/usecase/outbox"
	productuc "go-boilerplate/internal/usecase/product" // sample-api:line
)

func TestUsecaseModule_GraphIsValid(t *testing.T) {
	t.Parallel()

	// ユースケース層は機能追加で増える領域。各ユースケースの振る舞いは usecase 層のテストに任せ、
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

// sample-api:begin
func Test_provideProductUsecase(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("アップロード上限を設定から解決したユースケースを構築する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			txm := mock_tx.NewMockManager(ctrl)
			repo := mock_product.NewMockRepository(ctrl)
			categoryRepo := mock_category.NewMockRepository(ctrl)
			statusRepo := mock_status.NewMockRepository(ctrl)
			storage := mock_objectstorage.NewMockStorage(ctrl)
			authorizer := mock_authz.NewMockAuthorizer(ctrl)
			cfg := config.NewObjectStorageConfig(config.MockConfigForTest(t))
			tf := observability.NewNoopTracerFactory(t)

			got := provideProductUsecase(txm, repo, categoryRepo, statusRepo, storage, authorizer, cfg, tf)

			assert.Equal(t, productuc.New(txm, repo, categoryRepo, statusRepo, storage, authorizer,
				cfg.MaxUploadBytes(), tf), got)
		})
	})
}

// sample-api:end
