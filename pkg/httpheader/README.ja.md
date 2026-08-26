# httpheader

[English](README.md) | 日本語

HTTP ヘッダ名の分類。

## 公開 API

|関数|説明|
|---|---|
|`IsSensitive(name)`|そのヘッダが資格情報を運び、プロセス外へ転送してはならないものかを返す。|

## 補足

`IsSensitive` は大小文字と前後の空白を無視して判定するため、呼び出し側が `" Authorization"` や
`"AUTHORIZATION"` を渡しても `"authorization"` と同じ判定になる。単一の綴りだけを見ると、
資格情報がプロセス外へ出るかどうかを呼び出し側の書式が決めてしまう。

対象は `Authorization` / `Proxy-Authorization` / `Cookie` / `Set-Cookie` の 4 つに固定している。
答えるのは「そのヘッダが資格情報か」であって「その値が秘密か」ではない。個人情報を載せた
ヘッダの扱いは呼び出し側の判断であり、ここでは表現しない。
