package queue

import "testing"

func Test_StatsCollector_emitFailures(t *testing.T) {
	t.Parallel()
	t.Skip("StatsCollector.emitFailures は TestStatsCollector_Collect が失敗 counter の非出力/出力/累積を検証済み")
}
