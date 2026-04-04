package driver

import (
	"fmt"
	"net/url"
	"testing"

	"go-boilerplate/internal/config"

	"github.com/stretchr/testify/require"
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
	require.Equal(t, expected, actual)
}

func TestDSNWithTimeZone(t *testing.T) {
	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)
	osCfg := config.NewOperationSystemConfig(cfg)

	rawQuery := fmt.Sprintf("sslmode=%s&timezone=%s", dbCfg.SSLMode(), url.QueryEscape(osCfg.TimeZone()))

	expected := &url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(dbCfg.User(), dbCfg.Password()),
		Host:     fmt.Sprintf("%s:%d", dbCfg.Host(), dbCfg.Port()),
		Path:     dbCfg.DBName(),
		RawQuery: rawQuery,
	}

	actual := DSNWithTimeZone(dbCfg, osCfg)
	require.Equal(t, expected, actual)
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
	require.Equal(t, expected, actual)
}

func TestDSNWithTimeZoneString(t *testing.T) {
	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)
	osCfg := config.NewOperationSystemConfig(cfg)

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
	require.Equal(t, expected, actual)
}
