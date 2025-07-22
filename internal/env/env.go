package env

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Load は、環境変数を読み込む関数です。
func Load() error {
	env := os.Getenv("ENV")

	if env == "" {
		if err := godotenv.Load(".env/.env"); err != nil {
			return fmt.Errorf(".env/.env load failed : %v", err)
		}
		env = os.Getenv("ENV")
	}

	if err := godotenv.Load(fmt.Sprintf(".env/.env.%s", env)); err != nil {
		return fmt.Errorf(".env/.env.%s load failed : %v", env, err)
	}
	return nil
}
