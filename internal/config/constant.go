package config

// デプロイ環境ラベルの列挙。アプリ動作モード（*Mode）とは別軸で、値が一致するのは偶然であり連動させない。
const (
	// EnvLocal はローカル開発環境を表します。
	EnvLocal = "local"
	// EnvCI はCI環境を表します。
	EnvCI = "ci"
	// EnvTest はテスト環境を表します。
	EnvTest = "test"
	// EnvDevelopment は開発環境を表します（env ファイルの APP_ENV 実値）。
	EnvDevelopment = "dev"
	// EnvStaging はステージング環境を表します（env ファイルの APP_ENV 実値）。
	EnvStaging = "stg"
	// EnvProduction は本番環境を表します（env ファイルの APP_ENV 実値）。
	EnvProduction = "prd"
)

// アプリ動作モードの列挙（挙動切替に使用）。Env とは独立。
const (
	// DevelopmentMode は開発環境モードを表します。
	DevelopmentMode = "development"
	// ProductionMode は本番環境モードを表します。
	ProductionMode = "production"
)

// ログ出力レベルの列挙。APP_LOG_LEVEL が受理する値。
const (
	// LogLevelDebug はデバッグレベルを表します。
	LogLevelDebug = "debug"
	// LogLevelInfo は情報レベルを表します。
	LogLevelInfo = "info"
	// LogLevelWarn は警告レベルを表します。
	LogLevelWarn = "warn"
	// LogLevelError はエラーレベルを表します。
	LogLevelError = "error"
)

const (
	// MinPort は許可される最小ポート番号を表します。
	MinPort = 1
	// MaxPort は許可される最大ポート番号を表します。
	MaxPort = 65535
)
