package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

const envDir = "env"

// Load は、環境変数を読み込む関数です。
func Load() error {
	env := os.Getenv("ENV")

	if env == "" {
		base := filepath.Join(envDir, ".env")
		if err := godotenv.Load(base); err != nil {
			return fmt.Errorf("%w : %w", ErrFailedToLoadDefaultEnvFile, err)
		}
		env = os.Getenv("ENV")
	}

	if env == "" {
		return ErrEnvNotResolved
	}

	path := filepath.Join(envDir, ".env."+env)
	if err := godotenv.Load(path); err != nil {
		return fmt.Errorf("%w : %w", ErrFailedToLoadEnvFile, err)
	}
	return nil
}
