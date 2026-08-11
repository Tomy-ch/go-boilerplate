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

// ExtractFunctionName は、runtime.Func のフルネームからメソッド名のみを抽出します。
// 例:
//
//	"github.com/org/proj/foo/bar.(*Baz).Do"
//	  → "Do"
func ExtractFunctionName(full string) string {
	_, fn := splitFuncName(full)
	if fn == "" {
		return unknown
	}
	return fn
}

// ExtractPackageName は、runtime.Func のフルネームからパッケージ名を抽出します。
// ポインタ／値レシーバ・通常関数・ジェネリックのいずれの形式でも一貫して抽出できます。
// 例:
//
//	"github.com/org/proj/foo/bar.(*Baz).Do" → "bar"
//	"github.com/org/proj/foo/bar.Baz.Do"    → "bar"
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
