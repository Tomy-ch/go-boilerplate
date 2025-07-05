package env

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

func Load() error {
	if err := godotenv.Load(".env/.env"); err != nil {
		return fmt.Errorf(".env/.env load failed : %v", err)
	}

	if err := godotenv.Load(fmt.Sprintf(".env/.env.%s", os.Getenv("ENV"))); err != nil {
		return fmt.Errorf(".env/.env.%s load failed : %v", os.Getenv("ENV"), err)
	}
	return nil
}
