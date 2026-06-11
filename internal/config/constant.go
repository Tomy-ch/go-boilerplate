package config

// デプロイ環境ラベルの列挙。アプリ動作モード（*Mode）とは別軸で、値が一致するのは偶然であり連動させない。
const (
	// EnvLocal はローカル開発環境を表します。
	EnvLocal = "local"
	// EnvCI はCI環境を表します。
	EnvCI = "ci"
	// EnvTest はテスト環境を表します。
	EnvTest = "test"
	// EnvDevelopment は開発環境を表します。
	EnvDevelopment = "development"
	// EnvStaging はステージング環境を表します。
	EnvStaging = "staging"
	// EnvProduction は本番環境を表します。
	EnvProduction = "production"
)

// アプリ動作モードの列挙（挙動切替に使用）。Env とは独立。
const (
	// DevelopmentMode は開発環境モードを表します。
	DevelopmentMode = "development"
	// ProductionMode は本番環境モードを表します。
	ProductionMode = "production"
)

const (
	// MinPort は許可される最小ポート番号を表します。
	MinPort = 1
	// MaxPort は許可される最大ポート番号を表します。
	MaxPort = 65535
)
