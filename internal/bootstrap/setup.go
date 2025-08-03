// Package bootstrap は、アプリケーションの初期化と設定に関するパッケージです。
package bootstrap

import (
	"boilerplate-go/internal/appconfig"
	"boilerplate-go/internal/env"
)

// SetUpConfig は、Configを初期化するための関数です。
func SetUpConfig() (*appconfig.Config, error) {
	if err := env.Load(); err != nil {
		return nil, err
	}

	return appconfig.New()
}
