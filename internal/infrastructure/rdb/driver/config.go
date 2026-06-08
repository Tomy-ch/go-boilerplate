package driver

import (
	"fmt"
	"net/url"

	"go-boilerplate/internal/config"
)

// DSN は、データベースの接続URLを返します。
func DSN(dbCfg *config.DatabaseConfig) *url.URL {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(dbCfg.User(), dbCfg.Password()),
		Host:   fmt.Sprintf("%s:%d", dbCfg.Host(), dbCfg.Port()),
		Path:   dbCfg.DBName(),
	}

	q := u.Query()
	q.Set("sslmode", dbCfg.SSLMode())
	u.RawQuery = q.Encode()

	return u
}

// DSNWithTimeZone は、データベースの接続URLを返します。タイムゾーン情報をクエリパラメータに追加します。
func DSNWithTimeZone(db *config.DatabaseConfig, os *config.OperatingSystemConfig) *url.URL {
	u := DSN(db)
	q := u.Query()
	q.Set("timezone", os.TimeZone())
	u.RawQuery = q.Encode()
	return u
}

// DSNString は、DSNを文字列形式で返します。
func DSNString(dbCfg *config.DatabaseConfig) string {
	return DSN(dbCfg).String()
}

// DSNWithTimeZoneString は、DSNWithTimeZoneを文字列形式で返します。
func DSNWithTimeZoneString(db *config.DatabaseConfig, os *config.OperatingSystemConfig) string {
	return DSNWithTimeZone(db, os).String()
}
