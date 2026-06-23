package config

import (
	"bytes"
	"fmt"
	"os"

	rootenv "go-boilerplate"

	"github.com/joho/godotenv"
)

// Load は、埋め込まれた env ファイルから環境変数を設定します。
// 既に設定済みの環境変数は上書きしません（実行時注入を優先）。
func Load() error {
	b, err := rootenv.FS.ReadFile("env/.env")
	if err != nil {
		return fmt.Errorf("%w : %w", ErrFailedToLoadEnvFile, err)
	}

	kv, err := godotenv.Parse(bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("%w : %w", ErrFailedToParseConfig, err)
	}

	for k, v := range kv {
		if _, ok := os.LookupEnv(k); ok {
			continue
		}
		if err := os.Setenv(k, v); err != nil {
			return fmt.Errorf("%w : %w", ErrFailedToParseConfig, err)
		}
	}
	return nil
}
