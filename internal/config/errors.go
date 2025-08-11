package config

import (
	"errors"
	"fmt"
)

var (
	// errInvalidConfig は、コンフィグ設定に関するエラーを表します。
	errInvalidConfig = errors.New("config error")
	// ErrInvalidAppMode は、無効なアプリケーションモードに関するエラーを表します。
	ErrInvalidAppMode = fmt.Errorf(
		"invalid app mode, must which be one of %s or %s: %w",
		DevelopmentMode,
		ProductionMode,
		errInvalidConfig,
	)
	// ErrInvalidPortRange は、無効なポート範囲に関するエラーを表します。
	ErrInvalidPortRange = fmt.Errorf(
		"invalid port range, must be between %d and %d: %w",
		MinPort,
		MaxPort,
		errInvalidConfig,
	)
	// ErrEmptyAllowedOrigins は、許可されたオリジンが空であってはならないことを示すエラーです。
	ErrEmptyAllowedOrigins = fmt.Errorf("allowed origins must not be empty: %w", errInvalidConfig)
	// ErrHTTPOnlyAllowedForLocalhost は、HTTPのみのローカルホストにアクセス可能にするためのエラーを表します。
	ErrHTTPOnlyAllowedForLocalhost = fmt.Errorf(
		"http only localhost is allowed: %w",
		errInvalidConfig,
	)
	// ErrFailedToParseConfig は、環境変数のパースに失敗したことを示すエラーです。
	ErrFailedToParseConfig = fmt.Errorf(
		"failed to parse environment variables: %w",
		errInvalidConfig,
	)
	// ErrFailedToParseCIDR は、CIDRのパースに失敗したことを示すエラーです。
	ErrFailedToParseCIDR = fmt.Errorf(
		"failed to parse CIDR: %w",
		errInvalidConfig,
	)
)
