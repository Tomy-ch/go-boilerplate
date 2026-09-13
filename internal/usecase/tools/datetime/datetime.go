// Package datetime は、Usecase が外へ出す時刻のワイヤ表現を提供します。
//
// 綴り方を 1 箇所に集めるのは、同じ配送フレームの中で封筒と payload が違う読み方の時刻を並べないためです。
// 綴りに現れるオフセットは実行環境の TZ の関数なので、呼び出し側ごとに決めると表現がデプロイ先に依存します。
package datetime

import "time"

// FormatUTC は、時刻を UTC の RFC 3339（ナノ秒精度）で綴ります。
// ゼロ値も同じ規則で綴るため、値の有無の判定は呼び出し側が持ちます。
func FormatUTC(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}
