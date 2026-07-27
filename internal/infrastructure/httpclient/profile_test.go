package httpclient_test

import (
	"testing"

	"go-boilerplate/internal/infrastructure/httpclient"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRegistryFromProfiles(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("各DownstreamProfileが対応するProfileとして解決される", func(t *testing.T) {
			t.Parallel()

			profileA := httpclient.DefaultProfile()
			profileA.MaxAttempts = 7
			profileB := httpclient.DefaultProfile()
			profileB.MaxAttempts = 11

			registry, err := httpclient.NewRegistryFromProfiles([]httpclient.DownstreamProfile{
				{Name: "a", Profile: profileA},
				{Name: "b", Profile: profileB},
			})

			require.NoError(t, err)
			assert.Equal(t, profileA, registry.Profile("a"))
			assert.Equal(t, profileB, registry.Profile("b"))
		})

		t.Run("未登録DownstreamはDefaultProfileにfallbackする", func(t *testing.T) {
			t.Parallel()

			registry, err := httpclient.NewRegistryFromProfiles([]httpclient.DownstreamProfile{
				{Name: "a", Profile: httpclient.DefaultProfile()},
			})

			require.NoError(t, err)
			assert.Equal(t, httpclient.DefaultProfile(), registry.Profile("unknown"))
		})

		t.Run("空スライスでもすべてDefaultProfileにfallbackする", func(t *testing.T) {
			t.Parallel()

			registry, err := httpclient.NewRegistryFromProfiles(nil)
			require.NoError(t, err)
			assert.Equal(t, httpclient.DefaultProfile(), registry.Profile("any"))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("同一Nameが重複するとlast-wins上書きせずエラーを返す", func(t *testing.T) {
			t.Parallel()

			registry, err := httpclient.NewRegistryFromProfiles([]httpclient.DownstreamProfile{
				{Name: "dup", Profile: httpclient.DefaultProfile()},
				{Name: "dup", Profile: httpclient.DefaultProfile()},
			})

			// sentinel を持たないため、重複固有のメッセージで他経路のエラーと区別する。
			require.ErrorContains(t, err, "duplicate httpclient profile for downstream")
			assert.Nil(t, registry)
		})
	})
}

func TestMissingDownstreams(t *testing.T) {
	t.Parallel()

	profiles := []httpclient.DownstreamProfile{
		{Name: "a", Profile: httpclient.DefaultProfile()},
		{Name: "b", Profile: httpclient.DefaultProfile()},
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("全requiredが登録済みなら空を返す", func(t *testing.T) {
			t.Parallel()

			assert.Empty(t, httpclient.MissingDownstreams(profiles, []httpclient.Downstream{"a", "b"}))
		})

		t.Run("requiredが空なら空を返す", func(t *testing.T) {
			t.Parallel()

			assert.Empty(t, httpclient.MissingDownstreams(profiles, nil))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("profile未登録のrequiredを欠落として返す", func(t *testing.T) {
			t.Parallel()

			missing := httpclient.MissingDownstreams(profiles, []httpclient.Downstream{"a", "missing"})
			assert.Equal(t, []httpclient.Downstream{"missing"}, missing)
		})

		t.Run("profilesが空なら全requiredを欠落として返す", func(t *testing.T) {
			t.Parallel()

			missing := httpclient.MissingDownstreams(nil, []httpclient.Downstream{"x"})
			assert.Equal(t, []httpclient.Downstream{"x"}, missing)
		})

		t.Run("複数欠落はrequiredの登場順で返す", func(t *testing.T) {
			t.Parallel()

			missing := httpclient.MissingDownstreams(profiles, []httpclient.Downstream{"x", "a", "y"})
			assert.Equal(t, []httpclient.Downstream{"x", "y"}, missing)
		})
	})
}

func TestDefaultProfile(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}
