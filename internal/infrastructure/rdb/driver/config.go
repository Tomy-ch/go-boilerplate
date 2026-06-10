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
func DSNWithTimeZone(dbCfg *config.DatabaseConfig, osCfg *config.OperatingSystemConfig) *url.URL {
	u := DSN(dbCfg)
	q := u.Query()
	q.Set("timezone", osCfg.TimeZone())
	u.RawQuery = q.Encode()
	return u
}

// DSNString は、DSNを文字列形式で返します。
func DSNString(dbCfg *config.DatabaseConfig) string {
	return DSN(dbCfg).String()
}

// DSNStringWithoutPassword は、パスワードを含まない接続URL文字列を返します。
// 資格情報を引数に載せないため、パスワードは PGPASSWORD などで別途渡します。
func DSNStringWithoutPassword(dbCfg *config.DatabaseConfig) string {
	u := DSN(dbCfg)
	u.User = url.User(dbCfg.User())
	return u.String()
}

// DSNWithTimeZoneString は、DSNWithTimeZoneを文字列形式で返します。
func DSNWithTimeZoneString(dbCfg *config.DatabaseConfig, osCfg *config.OperatingSystemConfig) string {
	return DSNWithTimeZone(dbCfg, osCfg).String()
}
