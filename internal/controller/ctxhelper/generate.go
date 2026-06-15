// Package ctxhelper provides request-scoped context value helpers.
package ctxhelper

// 単純な値キーは scripts/genctxkey で生成できる（Authn は authn.go の手書きスロット方式）。

//go:generate go run ../../../scripts/genctxkey --name ErrorHandled --type bool --out .
//go:generate go run ../../../scripts/genctxkey --name Recovered --type bool --out .
