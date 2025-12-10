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
	// ErrInvalidReadHeaderTimeout は、ReadHeaderTimeoutが0以下であることを示すエラーです。
	ErrInvalidReadHeaderTimeout = fmt.Errorf(
		"invalid read header timeout, must be greater than 0: %w",
		errInvalidConfig,
	)
	// ErrInvalidReadTimeout は、ReadTimeoutが0以下であることを示すエラーです。
	ErrInvalidReadTimeout = fmt.Errorf(
		"invalid read timeout, must be greater than 0: %w",
		errInvalidConfig,
	)
	// ErrInvalidWriteTimeout は、WriteTimeoutが0以下であることを示すエラーです。
	ErrInvalidWriteTimeout = fmt.Errorf(
		"invalid write timeout, must be greater than 0: %w",
		errInvalidConfig,
	)
	// ErrInvalidIdleTimeout は、IdleTimeoutが0以下であることを示すエラーです。
	ErrInvalidIdleTimeout = fmt.Errorf(
		"invalid idle timeout, must be greater than 0: %w",
		errInvalidConfig,
	)
	// ErrReadHeaderTimeoutExceedsReadTimeout は、ReadHeaderTimeoutがReadTimeoutを超えていることを示すエラーです。
	ErrReadHeaderTimeoutExceedsReadTimeout = fmt.Errorf(
		"read header timeout exceeds read timeout: %w",
		errInvalidConfig,
	)
	// ErrInvalidSlowQueryWarnThreshold は、無効なスロークエリ警告閾値に関するエラーを表します。
	ErrInvalidSlowQueryWarnThreshold = fmt.Errorf(
		"invalid slow query warn threshold, must be greater than or equal to 0: %w",
		errInvalidConfig,
	)
	// ErrFailedToParseCIDR は、CIDRのパースに失敗したことを示すエラーです。
	ErrFailedToParseCIDR = fmt.Errorf(
		"failed to parse CIDR: %w",
		errInvalidConfig,
	)
)
