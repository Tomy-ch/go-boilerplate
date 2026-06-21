package config

import (
	"fmt"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"

	"golang.org/x/crypto/bcrypt"
)

var (
	// errInvalidConfig は、コンフィグ設定に関するエラーを表します。
	errInvalidConfig = xerrors.Wrap(apperror.ErrInvalidArgument, "config error")
	// ErrInvalidAppMode は、無効なアプリケーションモードに関するエラーを表します。
	ErrInvalidAppMode = xerrors.Wrap(
		errInvalidConfig,
		fmt.Sprintf("invalid app mode, must be one of %s or %s", DevelopmentMode, ProductionMode),
	)
	// ErrInvalidLogLevel は、無効なログレベルに関するエラーを表します。
	ErrInvalidLogLevel = xerrors.Wrap(
		errInvalidConfig,
		fmt.Sprintf(
			"invalid log level, must be one of %s, %s, %s or %s",
			LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError,
		),
	)
	// ErrInvalidPortRange は、無効なポート範囲に関するエラーを表します。
	ErrInvalidPortRange = xerrors.Wrap(
		errInvalidConfig,
		fmt.Sprintf("invalid port range, must be between %d and %d", MinPort, MaxPort),
	)
	// ErrEmptyAllowedOrigins は、許可されたオリジンが空であってはならないことを示すエラーです。
	ErrEmptyAllowedOrigins = xerrors.Wrap(errInvalidConfig, "allowed origins must not be empty")
	// ErrInvalidBcryptCost は、無効な bcrypt コストに関するエラーを表します。
	ErrInvalidBcryptCost = xerrors.Wrap(
		errInvalidConfig,
		fmt.Sprintf("invalid bcrypt cost, must be between %d and %d", bcrypt.MinCost, bcrypt.MaxCost),
	)
	// ErrHTTPOnlyAllowedForLocalhost は、HTTPのみのローカルホストにアクセス可能にするためのエラーを表します。
	ErrHTTPOnlyAllowedForLocalhost = xerrors.Wrap(
		errInvalidConfig,
		"http only localhost is allowed",
	)
	// ErrEnvNotResolved は、env/.env 読み込み後も ENV が解決できなかったことを示すエラーです。
	ErrEnvNotResolved = xerrors.Wrap(
		errInvalidConfig,
		"ENV is not set after loading env/.env",
	)
	// ErrFailedToLoadDefaultEnvFile は、デフォルトの env/.env 読み込みに失敗したことを示すエラーです。
	ErrFailedToLoadDefaultEnvFile = xerrors.Wrap(
		errInvalidConfig,
		"failed to load default env file",
	)
	// ErrFailedToLoadEnvFile は、ENV に対応する env/.env.<env> 読み込みに失敗したことを示すエラーです。
	ErrFailedToLoadEnvFile = xerrors.Wrap(
		errInvalidConfig,
		"failed to load env file",
	)
	// ErrFailedToParseConfig は、環境変数のパースに失敗したことを示すエラーです。
	ErrFailedToParseConfig = xerrors.Wrap(
		errInvalidConfig,
		"failed to parse environment variables",
	)
	// ErrInvalidReadHeaderTimeout は、ReadHeaderTimeoutが0以下であることを示すエラーです。
	ErrInvalidReadHeaderTimeout = xerrors.Wrap(
		errInvalidConfig,
		"invalid read header timeout, must be greater than 0",
	)
	// ErrInvalidReadTimeout は、ReadTimeoutが0以下であることを示すエラーです。
	ErrInvalidReadTimeout = xerrors.Wrap(
		errInvalidConfig,
		"invalid read timeout, must be greater than 0",
	)
	// ErrInvalidWriteTimeout は、WriteTimeoutが0以下であることを示すエラーです。
	ErrInvalidWriteTimeout = xerrors.Wrap(
		errInvalidConfig,
		"invalid write timeout, must be greater than 0",
	)
	// ErrInvalidIdleTimeout は、IdleTimeoutが0以下であることを示すエラーです。
	ErrInvalidIdleTimeout = xerrors.Wrap(
		errInvalidConfig,
		"invalid idle timeout, must be greater than 0",
	)
	// ErrReadHeaderTimeoutExceedsReadTimeout は、ReadHeaderTimeoutがReadTimeoutを超えていることを示すエラーです。
	ErrReadHeaderTimeoutExceedsReadTimeout = xerrors.Wrap(
		errInvalidConfig,
		"read header timeout exceeds read timeout",
	)
	// ErrInvalidDBPortRange は、データベースの無効なポート範囲に関するエラーを表します。
	ErrInvalidDBPortRange = xerrors.Wrap(
		errInvalidConfig,
		fmt.Sprintf("invalid database port range, must be between %d and %d", MinPort, MaxPort),
	)
	// ErrInvalidDBPingTimeout は、データベースの無効なPingタイムアウトに関するエラーを表します。
	ErrInvalidDBPingTimeout = xerrors.Wrap(
		errInvalidConfig,
		"invalid database ping timeout, must be greater than 0",
	)
	// ErrInvalidSlowQueryWarnThreshold は、無効なスロークエリ警告閾値に関するエラーを表します。
	ErrInvalidSlowQueryWarnThreshold = xerrors.Wrap(
		errInvalidConfig,
		"invalid slow query warn threshold, must be greater than or equal to 0",
	)
	// ErrInvalidExceedMaxConns は、データベースの最大接続数が過剰であることに関するエラーを表します。
	ErrInvalidExceedMaxConns = xerrors.Wrap(
		errInvalidConfig,
		"invalid database max connections, min connections must be less than or equal to max connections",
	)
	// ErrFailedToParseCIDR は、CIDRのパースに失敗したことを示すエラーです。
	ErrFailedToParseCIDR = xerrors.Wrap(
		errInvalidConfig,
		"failed to parse CIDR",
	)
	// ErrAuthConfigMissing は、認証設定が不足していることに関するエラーを表します。
	ErrAuthConfigMissing = xerrors.Wrap(
		errInvalidConfig,
		"invalid auth config, either cookie name or header name must be provided",
	)
)
