package module

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"go-boilerplate/internal/infrastructure/httpclient"
	authzbd "go-boilerplate/internal/usecase/boundary/authz"
	"go-boilerplate/internal/usecase/boundary/clock"
	objectstoragebd "go-boilerplate/internal/usecase/boundary/objectstorage"
	outboxbndry "go-boilerplate/internal/usecase/boundary/outbox"
	publisherbd "go-boilerplate/internal/usecase/boundary/publisher"
)

func TestInfrastructureModule_GraphIsValid(t *testing.T) {
	t.Parallel()

	// 各 concern（persistence / clock / httpclient / webapi / auth / authz）の
	// 個別検証は各 concern の *_test.go に切り出し、ここでは集約された InfrastructureModule 全体が
	// 欠落なく結線されることを確認する。outbox_publisher は relay 専用のため OutboxRelayModule 側で検証する。
	// SQL や実 DB 挙動は infra 層のテストに任せる。
	opts := append(commonDeps(), InfrastructureModule())
	validateGraph(t, opts...)
}

func TestInfrastructureModule(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("集約した各 concern の代表型を単体で提供する", func(t *testing.T) {
			t.Parallel()

			var (
				store   outboxbndry.Store       // persistence
				clk     clock.Clock             // clock
				client  httpclient.Client       // httpclient
				storage objectstoragebd.Storage // objectstorage
				authz   authzbd.Authorizer      // authz
			)

			validateGraph(t, append(commonDeps(), InfrastructureModule(),
				fx.Populate(&store, &clk, &client, &storage, &authz))...)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("relay 専用の outbox publisher は含まず解決できない", func(t *testing.T) {
			t.Parallel()

			// publisher は非標準の httpclient profile を寄与するため、relay 以外へ漏らさない。
			var publisher publisherbd.Publisher

			opts := append(commonDeps(), InfrastructureModule(), fx.Populate(&publisher), fx.NopLogger)
			require.Error(t, fx.ValidateApp(opts...))
		})
	})
}
