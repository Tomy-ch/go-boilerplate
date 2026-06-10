package driver

import (
	"fmt"
	"net/url"

	"go-boilerplate/internal/config"
)

// buildDSN は、sslmode と追加クエリパラメータを一度の Encode で確定して接続URLを組み立てます。
func buildDSN(dbCfg *config.DatabaseConfig, extra url.Values) *url.URL {
	q := url.Values{}
	q.Set("sslmode", dbCfg.SSLMode())
	for key, values := range extra {
		for _, v := range values {
			q.Set(key, v)
		}
	}

	return &url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(dbCfg.User(), dbCfg.Password()),
		Host:     fmt.Sprintf("%s:%d", dbCfg.Host(), dbCfg.Port()),
		Path:     dbCfg.DBName(),
		RawQuery: q.Encode(),
	}
}

// DSN は、データベースの接続URLを返します。
func DSN(dbCfg *config.DatabaseConfig) *url.URL {
	return buildDSN(dbCfg, nil)
}

// DSNWithTimeZone は、データベースの接続URLを返します。タイムゾーン情報をクエリパラメータに追加します。
func DSNWithTimeZone(dbCfg *config.DatabaseConfig, osCfg *config.OperatingSystemConfig) *url.URL {
	return buildDSN(dbCfg, url.Values{"timezone": {osCfg.TimeZone()}})
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
