package driver

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"

	"go-boilerplate/internal/config"
)

func TestDSN(t *testing.T) {
	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)

	expected := &url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(dbCfg.User(), dbCfg.Password()),
		Host:     fmt.Sprintf("%s:%d", dbCfg.Host(), dbCfg.Port()),
		Path:     dbCfg.DBName(),
		RawQuery: fmt.Sprintf("sslmode=%s", dbCfg.SSLMode()),
	}

	actual := DSN(dbCfg)
	assert.Equal(t, expected, actual)
}

func TestDSNWithTimeZone(t *testing.T) {
	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)
	osCfg := config.NewOperatingSystemConfig(cfg)

	rawQuery := fmt.Sprintf("sslmode=%s&timezone=%s", dbCfg.SSLMode(), url.QueryEscape(osCfg.TimeZone()))

	expected := &url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(dbCfg.User(), dbCfg.Password()),
		Host:     fmt.Sprintf("%s:%d", dbCfg.Host(), dbCfg.Port()),
		Path:     dbCfg.DBName(),
		RawQuery: rawQuery,
	}

	actual := DSNWithTimeZone(dbCfg, osCfg)
	assert.Equal(t, expected, actual)
}

func TestDSNString(t *testing.T) {
	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)

	urlCfg := &url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(dbCfg.User(), dbCfg.Password()),
		Host:     fmt.Sprintf("%s:%d", dbCfg.Host(), dbCfg.Port()),
		Path:     dbCfg.DBName(),
		RawQuery: fmt.Sprintf("sslmode=%s", dbCfg.SSLMode()),
	}
	expected := urlCfg.String()

	actual := DSNString(dbCfg)
	assert.Equal(t, expected, actual)
}

func TestDSNWithTimeZoneString(t *testing.T) {
	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)
	osCfg := config.NewOperatingSystemConfig(cfg)

	rawQuery := fmt.Sprintf("sslmode=%s&timezone=%s", dbCfg.SSLMode(), url.QueryEscape(osCfg.TimeZone()))

	urlCfg := &url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(dbCfg.User(), dbCfg.Password()),
		Host:     fmt.Sprintf("%s:%d", dbCfg.Host(), dbCfg.Port()),
		Path:     dbCfg.DBName(),
		RawQuery: rawQuery,
	}
	expected := urlCfg.String()

	actual := DSNWithTimeZoneString(dbCfg, osCfg)
	assert.Equal(t, expected, actual)
}
