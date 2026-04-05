// Package fnmeta は、関数名やパッケージ名の抽出に関するユーティリティを提供します。
package fnmeta

import (
	"strings"
)

const undefined = "unknown"

// splitFuncName は、runtime.Func のフルネームを
// 「パッケージ/レシーバ部」と「メソッド名」に分解します。
// 例:
//
//	"github.com/org/proj/internal/usecase/user.(*UserUsecase).GetUser"
//	  → lhs: "user.(*UserUsecase)", rhs: "GetUser"
func splitFuncName(full string) (string, string) {
	if full == "" {
		return undefined, ""
	}

	// パス部分を除去: internal/usecase/user.(*UserUsecase).GetUser
	if idx := strings.LastIndex(full, "/"); idx >= 0 && idx+1 < len(full) {
		full = full[idx+1:]
	}

	// 最後の "." で分割: lhs = "user.(*UserUsecase)", rhs = "GetUser"
	if idx := strings.LastIndex(full, "."); idx >= 0 && idx+1 < len(full) {
		return full[:idx], full[idx+1:]
	}

	// "." が見つからない場合は全部を lhs とみなす
	return full, ""
}

// ExtractFunctionName は、runtime.Func のフルネームから span 名として使いやすい
// メソッド名のみを抽出します。
// 例:
//
//	"github.com/org/proj/internal/usecase/user.(*UserUsecase).GetUser"
//	  → "GetUser"
func ExtractFunctionName(full string) string {
	_, fn := splitFuncName(full)
	if fn == "" {
		return undefined
	}
	return fn
}

// ExtractPackageName は、runtime.Func のフルネームからパッケージ名を抽出します。
// 例:
//
//	"github.com/org/proj/internal/usecase/user.(*UserUsecase).GetUser"
//	  → "user"
func ExtractPackageName(full string) string {
	lhs, _ := splitFuncName(full)
	if lhs == undefined {
		return undefined
	}

	if idx := strings.Index(lhs, "("); idx >= 0 {
		lhs = strings.TrimSuffix(lhs[:idx], ".")
	}

	return lhs
}
