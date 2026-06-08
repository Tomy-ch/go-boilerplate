package config

import "time"

// NewTimeLocation は、新しい時間ロケーションを作成して返します。
func NewTimeLocation(osCfg *OperatingSystemConfig) (*time.Location, error) {
	return time.LoadLocation(osCfg.TimeZone())
}
