package module

import (
	"go.uber.org/fx"
)

// webapiModule は、外部 Web API クライアント（gateway）を提供するfx.Moduleです。
func webapiModule() fx.Option {
	return fx.Module("webapi") // gateway を足すときは、コンストラクタを fx.Provide へ、HTTP クライアントの
	// プロファイルと必須 downstream をそれぞれのグループ提供子へ渡す。
}
