package config

import (
	"bytes"
	"os"

	rootenv "go-boilerplate"
	"go-boilerplate/pkg/xerrors"

	"github.com/joho/godotenv"
)

// Load は、埋め込まれた env ファイルから環境変数を設定します。
// 既に設定済みの環境変数は上書きしません（実行時注入を優先）。
func Load() error {
	b, err := rootenv.FS.ReadFile("env/.env")
	if err != nil {
		return xerrors.Join(ErrFailedToLoadEnvFile, err)
	}

	kv, err := godotenv.Parse(bytes.NewReader(b))
	if err != nil {
		return xerrors.Join(ErrFailedToParseConfig, err)
	}

	for k, v := range kv {
		if _, ok := os.LookupEnv(k); ok {
			continue
		}
		if err := os.Setenv(k, v); err != nil {
			return xerrors.Join(ErrFailedToParseConfig, err)
		}
	}
	return nil
}
