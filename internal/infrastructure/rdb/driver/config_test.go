package driver

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"

	"go-boilerplate/internal/config"
)

func TestDSN(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)

	expected := &url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(dbCfg.User(), dbCfg.Password()),
		Host:     fmt.Sprintf("%s:%d", dbCfg.Host(), dbCfg.Port()),
		Path:     dbCfg.DBName(),
		RawQuery: "sslmode=" + dbCfg.SSLMode(),
	}

	actual := DSN(dbCfg)
	assert.Equal(t, expected, actual)
}

func TestDSNWithTimeZone(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)

	urlCfg := &url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(dbCfg.User(), dbCfg.Password()),
		Host:     fmt.Sprintf("%s:%d", dbCfg.Host(), dbCfg.Port()),
		Path:     dbCfg.DBName(),
		RawQuery: "sslmode=" + dbCfg.SSLMode(),
	}
	expected := urlCfg.String()

	actual := DSNString(dbCfg)
	assert.Equal(t, expected, actual)
}

func TestDSNStringWithoutPassword(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)

	urlCfg := &url.URL{
		Scheme:   "postgres",
		User:     url.User(dbCfg.User()),
		Host:     fmt.Sprintf("%s:%d", dbCfg.Host(), dbCfg.Port()),
		Path:     dbCfg.DBName(),
		RawQuery: "sslmode=" + dbCfg.SSLMode(),
	}
	expected := urlCfg.String()

	actual := DSNStringWithoutPassword(dbCfg)
	assert.Equal(t, expected, actual)
	// パスワードが含まれないこと。
	assert.NotContains(t, actual, dbCfg.Password())
}

func TestDSNWithTimeZoneString(t *testing.T) {
	t.Parallel()

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
