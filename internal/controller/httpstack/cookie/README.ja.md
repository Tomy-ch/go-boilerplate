# cookie

[English](README.md) | 日本語

Set-Cookie ヘッダのセキュリティ属性（Secure / HttpOnly / SameSite / Path / Domain / Max-Age）を強制するミドルウェアです。

## 公開 API

|関数 / 型|説明|
|---|---|
|`NewSecurityCookie(p)`|`SecureCookieConfig` から `SecurityCookie` 設定を生成|
|`Middleware(cfg)`|Set-Cookie ヘッダを書き換える Echo ミドルウェアを返す|
|`RewriteSetCookie(raw)`|生の Set-Cookie ヘッダ文字列をセキュリティポリシーに基づき書き換え|
|`SecurityCookie`|Cookie 属性強制の設定構造体|
