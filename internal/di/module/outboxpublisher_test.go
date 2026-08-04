package module

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"go-boilerplate/internal/infrastructure/httpclient"
	outboxpublisher "go-boilerplate/internal/infrastructure/publisher"
	publisherbd "go-boilerplate/internal/usecase/boundary/publisher"
)

func Test_outboxPublisherModule_GraphIsValid(t *testing.T) {
	t.Parallel()

	// outbox publish 先（HTTP）は httpclient.Client / TracerFactory に依存するため、
	// substrate（clock + httpclient）も併せて配線し、グラフが欠落なく解決することを確認する。
	opts := append(commonDeps(), clockModule(), httpClientModule(), outboxPublisherModule())
	validateGraph(t, opts...)
}

func Test_outboxPublisherModule(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("publish 境界の Publisher を提供する", func(t *testing.T) {
			t.Parallel()

			// 送信先の解決は判別子ごとの分岐内で行うため、Endpoint はグラフに露出しない。
			var publisher publisherbd.Publisher

			validateGraph(t, append(commonDeps(), clockModule(), httpClientModule(), outboxPublisherModule(),
				fx.Populate(&publisher))...)
		})

		t.Run("outbox 宛の非標準 profile を required と対で寄与する", func(t *testing.T) {
			t.Parallel()

			// required だけ寄与して profile を欠くと registry 構築が起動時に失敗する。
			profiles := collectGroup[httpclient.DownstreamProfile](t, `group:"httpclient_profiles"`, outboxPublisherModule())
			required := collectGroup[httpclient.Downstream](t, `group:"required_downstreams"`, outboxPublisherModule())

			assert.Equal(t, []httpclient.Downstream{outboxpublisher.RequiredDownstream()}, required)
			assert.Empty(t, httpclient.MissingDownstreams(profiles, required))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未配線では Publisher が解決できずグラフ検証に失敗する", func(t *testing.T) {
			t.Parallel()

			var publisher publisherbd.Publisher

			opts := append(commonDeps(), clockModule(), httpClientModule(),
				fx.Populate(&publisher), fx.NopLogger)
			require.Error(t, fx.ValidateApp(opts...))
		})
	})
}
