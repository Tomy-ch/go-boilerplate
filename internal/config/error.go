package config

import (
	"fmt"

	"boilerplate-go/internal/apperror"
)

var (
	// errInvalidConfig は、コンフィグ設定に関するエラーを表します。
	errInvalidConfig = fmt.Errorf("config error: %w", apperror.ErrInvalidArgument)
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
	// ErrReadHeaderTimeoutExceedsReadTimeout は、ReadHeaderTimeoutがReadTimeoutを超えていることを示すエラーです。
	ErrReadHeaderTimeoutExceedsReadTimeout = fmt.Errorf(
		"read header timeout exceeds read timeout: %w",
		errInvalidConfig,
	)
	// ErrFailedToParseCIDR は、CIDRのパースに失敗したことを示すエラーです。
	ErrFailedToParseCIDR = fmt.Errorf(
		"failed to parse CIDR: %w",
		errInvalidConfig,
	)
)
