package config

// SetUpConfig は、埋め込み env ファイルを環境変数へ反映したうえで Config を構築して返します。
// 既に設定済みの環境変数は上書きしません（実行時注入を優先）。
func SetUpConfig() (*Config, error) {
	if err := Load(); err != nil {
		return nil, err
	}

	return New()
}
