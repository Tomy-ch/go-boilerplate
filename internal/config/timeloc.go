package config

import "time"

// NewTimeLocation は、OS 設定のタイムゾーンから time.Location を構築します。
// IANA タイムゾーンデータベースで解決できない値が設定されている場合はエラーを返します。
func NewTimeLocation(osCfg *OperatingSystemConfig) (*time.Location, error) {
	return time.LoadLocation(osCfg.TimeZone())
}
