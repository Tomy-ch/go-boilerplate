package config

import (
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewConfig(t *testing.T) {
	expectedEnv := "test"
	expectedAppMode := "development"
	expectedHost := "localhost"
	expectedPort := 8080
	expectedAllowedOrigins := "http://localhost,https://example.com"

	t.Run("正常系", func(t *testing.T) {
		t.Run("configに必要な環境変数が全て設定されている場合", func(t *testing.T) {
			t.Setenv("ENV", expectedEnv)
			t.Setenv("APP_MODE", expectedAppMode)
			t.Setenv("HOST", expectedHost)
			t.Setenv("PORT", strconv.Itoa(expectedPort))
			t.Setenv("ALLOWED_ORIGINS", expectedAllowedOrigins)

			expected := &Config{
				environment: environment{
					serverEnv: expectedEnv,
					appMode:   expectedAppMode,
				},
				server: server{
					host:           expectedHost,
					port:           expectedPort,
					allowedOrigins: strings.Split(expectedAllowedOrigins, ","),
				},
			}

			actual, err := New()

			assert.NoError(t, err)
			assert.Equal(t, expected, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Run("configに必要な環境変数が不足している場合", func(t *testing.T) {
			t.Setenv("ENV", expectedEnv)

			actual, err := New()
			assert.Nil(t, actual)
			assert.ErrorContains(t, err, "APP_MODE")
		})

		t.Run("バリデート結果がエラーの場合", func(t *testing.T) {
			t.Setenv("ENV", expectedEnv)
			t.Setenv("APP_MODE", "invalid_env")
			t.Setenv("HOST", expectedHost)
			t.Setenv("PORT", strconv.Itoa(expectedPort))
			t.Setenv("ALLOWED_ORIGINS", expectedAllowedOrigins)

			actual, err := New()
			assert.Nil(t, actual)
			assert.ErrorIs(t, err, ErrInvalidAppMode)
		})
	})
}

func TestGetterMethods(t *testing.T) {
	t.Parallel()

	expectedEnv := "test"
	expectedAppMode := "development"
	expectedHost := "localhost"
	expectedPort := 8080
	expectedAllowedOrigins := "http://localhost,https://example.com"

	cfg := &Config{
		environment: environment{
			serverEnv: expectedEnv,
			appMode:   expectedAppMode,
		},
		server: server{
			host:           expectedHost,
			port:           expectedPort,
			allowedOrigins: strings.Split(expectedAllowedOrigins, ","),
		},
	}

	t.Run("ServerHost", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, expectedHost, cfg.ServerHost())
	})

	t.Run("ServerPort", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, expectedPort, cfg.ServerPort())
	})

	t.Run("AllowedOrigins", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, strings.Split(expectedAllowedOrigins, ","), cfg.AllowedOrigins())
	})

	t.Run("ServerEnv", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, expectedEnv, cfg.ServerEnv())
	})

	t.Run("AppMode", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, expectedAppMode, cfg.AppMode())
	})
}

func Test_validateConfig(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		cfg := verifiedConfigLoader()

		err := validateConfig(cfg)
		assert.NoError(t, err)
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()
		t.Run("無効なポート番号", func(t *testing.T) {
			t.Parallel()
			cfg := verifiedConfigLoader()
			cfg.Server.Port = MaxPort + 1 // 無効なポート番号

			err := validateConfig(cfg)
			assert.ErrorIs(t, err, ErrInvalidPortRange)
		})
		t.Run("無効なアプリケーションモード", func(t *testing.T) {
			t.Parallel()
			cfg := verifiedConfigLoader()
			cfg.Environment.AppMode = "invalid_mode" // 無効なアプリケーションモード

			err := validateConfig(cfg)
			assert.ErrorIs(t, err, ErrInvalidAppMode)
		})

		t.Run("ローカルホスト以外のHTTPプロトコルのアドレスの場合", func(t *testing.T) {
			t.Parallel()
			cfg := verifiedConfigLoader()
			cfg.Server.AllowedOrigins = []string{"http://example.com"} // HTTPのみのローカルホストが含まれていない

			err := validateConfig(cfg)
			assert.ErrorIs(t, err, ErrHTTPOnlyAllowedForLocalhost)
		})
	})
}

func TestIsAppEnvProduction(t *testing.T) {
	t.Parallel()
	t.Run("本番環境モードの場合", func(t *testing.T) {
		t.Parallel()
		cfg := Config{}
		cfg.environment.appMode = ProductionMode
		assert.True(t, cfg.IsAppEnvProduction())
	})

	t.Run("開発環境モードの場合", func(t *testing.T) {
		t.Parallel()
		cfg := Config{}
		cfg.environment.appMode = DevelopmentMode
		assert.False(t, cfg.IsAppEnvProduction())
	})
}

func TestIsAppEnvDevelopment(t *testing.T) {
	t.Parallel()
	t.Run("開発環境モードの場合", func(t *testing.T) {
		t.Parallel()
		cfg := Config{}
		cfg.environment.appMode = DevelopmentMode
		assert.True(t, cfg.IsAppEnvDevelopment())
	})

	t.Run("本番環境モードの場合", func(t *testing.T) {
		t.Parallel()
		cfg := Config{}
		cfg.environment.appMode = ProductionMode
		assert.False(t, cfg.IsAppEnvDevelopment())
	})
}

func verifiedConfigLoader() ConfigLoader {
	return ConfigLoader{
		Server: Server{
			Host:           "localhost",
			Port:           8080,
			AllowedOrigins: []string{"http://localhost", "https://example.com"},
		},
		Environment: Environment{
			ServerEnv: "test",
			AppMode:   DevelopmentMode,
		},
	}
}
