// Package fnmeta は、関数名やパッケージ名の抽出に関するユーティリティを提供します。
package fnmeta

import (
	"strings"
)

const unknown = "unknown"

// splitFuncName は、runtime.Func のフルネームを
// 「パッケージ/レシーバ部」と「メソッド名」に分解します。
// 例:
//
//	"github.com/org/proj/internal/usecase/user.(*UserUsecase).GetUser"
//	  → lhs: "user.(*UserUsecase)", rhs: "GetUser"
func splitFuncName(full string) (string, string) {
	if full == "" {
		return unknown, ""
	}

	if idx := strings.LastIndex(full, "/"); idx >= 0 {
		full = full[idx+1:]
	}

	if idx := strings.LastIndex(full, "."); idx >= 0 {
		return full[:idx], full[idx+1:]
	}

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
		return unknown
	}
	return fn
}

// ExtractPackageName は、runtime.Func のフルネームからパッケージ名を抽出します。
// パッケージ名は lhs の先頭セグメント（最初の "." より前）なので、ポインタ／値
// レシーバ・通常関数・ジェネリックのいずれの形式でも一貫して抽出できる。
// 例:
//
//	"github.com/org/proj/internal/usecase/user.(*UserUsecase).GetUser" → "user"
//	"github.com/org/proj/internal/usecase/user.UserUsecase.GetUser"    → "user"
func ExtractPackageName(full string) string {
	lhs, fn := splitFuncName(full)
	// 分割できない（"." が無い）入力は抽出不能として ExtractFunctionName と
	// 同様に unknown へ正規化する。
	if lhs == unknown || fn == "" {
		return unknown
	}

	if before, _, ok := strings.Cut(lhs, "."); ok {
		return before
	}

	return lhs
}
