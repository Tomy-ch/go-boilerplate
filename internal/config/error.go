package config

import (
	"fmt"

	"go-boilerplate/internal/apperror"

	"golang.org/x/crypto/bcrypt"
)

var (
	// errInvalidConfig は、コンフィグ設定に関するエラーを表します。
	errInvalidConfig = fmt.Errorf("config error: %w", apperror.ErrInvalidArgument)
	// ErrInvalidAppMode は、無効なアプリケーションモードに関するエラーを表します。
	ErrInvalidAppMode = fmt.Errorf(
		"invalid app mode, must be one of %s or %s: %w",
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
	// ErrInvalidBcryptCost は、無効な bcrypt コストに関するエラーを表します。
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
	// ErrEnvNotResolved は、env/.env 読み込み後も ENV が解決できなかったことを示すエラーです。
	ErrEnvNotResolved = fmt.Errorf(
		"ENV is not set after loading env/.env: %w",
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
	// ErrAuthConfigMissing は、認証設定が不足していることに関するエラーを表します。
	ErrAuthConfigMissing = fmt.Errorf(
		"invalid auth config, either cookie name or header name must be provided: %w",
		errInvalidConfig,
	)
)
