package config

import (
	"fmt"

	"boilerplate-go/internal/apperror"

	"golang.org/x/crypto/bcrypt"
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
	//
	ErrInvalidBcryptCost = fmt.Errorf(
		"invalid bcrypt cost, must be between %d and %d: %w",
		bcrypt.MinCost,
		bcrypt.MaxCost,
		errInvalidConfig,
	)
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
	// ErrInvalidDBPortRange は、データベースの無効なポート範囲に関するエラーを表します。
	ErrInvalidDBPortRange = fmt.Errorf(
		"invalid database port range, must be between %d and %d: %w",
		MinPort,
		MaxPort,
		errInvalidConfig,
	)
	// ErrInvalidDBPingTimeout は、データベースの無効なPingタイムアウトに関するエラーを表します。
	ErrInvalidDBPingTimeout = fmt.Errorf(
		"invalid database ping timeout, must be greater than 0: %w",
		errInvalidConfig,
	)
	// ErrInvalidSlowQueryWarnThreshold は、無効なスロークエリ警告閾値に関するエラーを表します。
	ErrInvalidSlowQueryWarnThreshold = fmt.Errorf(
		"invalid slow query warn threshold, must be greater than or equal to 0: %w",
		errInvalidConfig,
	)
	// ErrInvalidExceedMaxConns は、データベースの最大接続数が過剰であることに関するエラーを表します。
	ErrInvalidExceedMaxConns = fmt.Errorf(
		"invalid database max connections, min connections must be less than or equal to max connections: %w",
		errInvalidConfig,
	)
	// ErrFailedToParseCIDR は、CIDRのパースに失敗したことを示すエラーです。
	ErrFailedToParseCIDR = fmt.Errorf(
		"failed to parse CIDR: %w",
		errInvalidConfig,
	)
	// ErrInvalidIPRateLimitRequests は、無効なIPレートリミットのリクエスト数に関するエラーを表します。
	ErrInvalidIPRateLimitRequests = fmt.Errorf(
		"invalid IP rate limit requests, must be greater than 0: %w",
		errInvalidConfig,
	)
	// ErrInvalidIPRateLimitPer は、無効なIPレートリミットの期間に関するエラーを表します。
	ErrInvalidIPRateLimitPer = fmt.Errorf(
		"invalid IP rate limit per, must be greater than 0: %w",
		errInvalidConfig,
	)
	// ErrInvalidIPRateLimitBurst は、無効なIPレートリミットのバーストに関するエラーを表します。
	ErrInvalidIPRateLimitBurst = fmt.Errorf(
		"invalid IP rate limit burst, must be greater than or equal to 0: %w",
		errInvalidConfig,
	)
	// ErrInvalidIPRateLimitTTL は、無効なIPレートリミットのTTLに関するエラーを表します。
	ErrInvalidIPRateLimitTTL = fmt.Errorf(
		"invalid IP rate limit TTL, must be greater than 0: %w",
		errInvalidConfig,
	)
	// ErrInvalidIPRateLimitCleanupInterval は、無効なIPレートリミットのクリーンアップ間隔に関するエラーを表します。
	ErrInvalidIPRateLimitCleanupInterval = fmt.Errorf(
		"invalid IP rate limit cleanup interval, must be greater than 0: %w",
		errInvalidConfig,
	)
)
