# ptr

[English](README.md) | 日本語

ジェネリクスを利用したポインタ操作ユーティリティです。

## 公開 API

|関数|説明|
|---|---|
|`To[T](v T) *T`|値からポインタを生成|
|`Copy[T](v *T) *T`|ポインタのコピー（nil 入力時は nil を返す）|
|`Deref[T](p *T, fallback T) T`|ポインタの参照外し（nil の場合は fallback を返す）|

## 注意点

Go 1.18 以降が必要（ジェネリクス）。
