package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

func New() (*Config, error) {
	cfg := Config{}

	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return nil, fmt.Errorf("env parse failed : %v", err)
	}
	return &cfg, nil
}
