package httpclient_test

import (
	"testing"

	"go-boilerplate/internal/infrastructure/httpclient"

	"github.com/stretchr/testify/assert"
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

			registry := httpclient.NewRegistryFromProfiles([]httpclient.DownstreamProfile{
				{Name: "a", Profile: profileA},
				{Name: "b", Profile: profileB},
			})

			assert.Equal(t, profileA, registry.Profile("a"))
			assert.Equal(t, profileB, registry.Profile("b"))
		})

		t.Run("未登録DownstreamはDefaultProfileにfallbackする", func(t *testing.T) {
			t.Parallel()

			registry := httpclient.NewRegistryFromProfiles([]httpclient.DownstreamProfile{
				{Name: "a", Profile: httpclient.DefaultProfile()},
			})

			assert.Equal(t, httpclient.DefaultProfile(), registry.Profile("unknown"))
		})

		t.Run("空スライスでもすべてDefaultProfileにfallbackする", func(t *testing.T) {
			t.Parallel()

			registry := httpclient.NewRegistryFromProfiles(nil)
			assert.Equal(t, httpclient.DefaultProfile(), registry.Profile("any"))
		})
	})
}
