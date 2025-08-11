package config

// SetUpConfig は、Configを初期化するための関数です。
func SetUpConfig() (*Config, error) {
	if err := Load(); err != nil {
		return nil, err
	}

	return New()
}
