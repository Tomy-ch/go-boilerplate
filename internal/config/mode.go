package config

const (
	// ExecutionModeServer は、サーバーモードを表します。
	ExecutionModeServer ExecutionMode = "server"
	// ExecutionModeJob は、ジョブモードを表します。
	ExecutionModeJob ExecutionMode = "job"
)

// ExecutionMode は、アプリケーションの実行モードを表す型です。
type ExecutionMode string
