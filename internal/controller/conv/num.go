package conv

// Int32 は、usecase の DTO の int をレスポンスの int32 へ変換します。
// 呼び出し側が int32 の範囲に収まると保証できる値（32bit 整数幅で永続化される DB 列由来の数量等）に
// 限って使用します。範囲外になり得る値には使用しないでください。
func Int32(v int) int32 {
	//nolint:gosec // G115: 呼び出し側が int32 範囲を保証する値のみを渡すためオーバーフローしません
	return int32(v)
}
