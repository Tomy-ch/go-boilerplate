package observability

import "runtime"

// skipCallerFramesForLayerTracer は runtime.Caller のスキップ段数。getCallerFullName 自身(1) と
// 直接呼び出し元(1) を飛ばす内訳。呼び出し元は本関数を直接呼ぶこと（間にラッパを挟むとずれる）。
const skipCallerFramesForLayerTracer = 2

// getCallerFullName は、本関数の直接呼び出し元のフル関数名を返す（取得失敗時は空文字）。
func getCallerFullName() string {
	pc, _, _, ok := runtime.Caller(skipCallerFramesForLayerTracer)
	if !ok {
		return ""
	}
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return ""
	}
	return fn.Name()
}
