package observability

import "runtime"

const skipCallerFramesForLayerTracer = 2

// getCallerFullName は、スタックの n 番目の呼び出し元のフル関数名を返す。
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
