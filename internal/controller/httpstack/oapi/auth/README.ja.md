# oapi/auth

[English](README.md) | 日本語

`Authorization` ヘッダーから Bearer トークンを抽出し、boundary の `Authenticator` で検証し、結果をリクエストコンテキスト（authn スロット）に格納する OpenAPI 認証関数です。Cookie ベースの抽出はサポートしません（Bearer / リソースサーバーモデル）。

## トークン抽出フロー

```mermaid
flowchart TB
    Start["リクエスト"]
    Header{"HeaderName 設定あり?"}
    IsAuth{"Authorization ヘッダー + AllowedHeaderBearer?"}
    StripBearer["'Bearer ' プレフィックス除去 → scheme=Bearer"]
    RawHeader["生のヘッダー値を使用 → scheme 空"]
    NoToken["トークンなし"]
    Credential["NewCredential(scheme, token)"]
    Authenticate["authenticator.Authenticate(ctx, credential)"]
    StoreAuthn["ctxhelper.SetAuthn(req.Context(), authn)"]

    Start --> Header
    Header -- yes --> IsAuth
    IsAuth -- yes --> StripBearer --> Credential
    IsAuth -- no --> RawHeader --> Credential
    Header -- no --> NoToken
    Credential --> Authenticate --> StoreAuthn
```

### 抽出ルール

1. **Header** — `AuthConfig.HeaderName()` が設定されている場合、そのヘッダーから抽出（デフォルト `Authorization`）
2. **Bearer プレフィックス** — `AllowedHeaderBearer` が true でヘッダーが `Authorization` の場合、`Bearer` プレフィックス（末尾のスペース込み）を除去。credential のスキームは `Bearer` になる
3. トークンが取得できない場合は `ErrUnauthorizedTokenNotProvided` を返す

### 認証ステップ

1. `Authorization` ヘッダーから Bearer トークンを抽出（上記ルールに従う）
2. スキームとトークンから `boundary/auth.Credential` を生成
3. `authenticator.Authenticate(ctx, credential)` を呼び出して `Authn` を取得
4. `ctxhelper.SetAuthn()` でリクエストコンテキストに `Authn` を格納（スロットは上流の `oapi.Middleware` が `ctxhelper.WithAuthn` で仕込む）。スロットが無ければ `ErrAuthnSlotNotFound` を返す

ハンドラコードでは `ctxhelper.GetAuthn()` で `Authn` を取得できます。

## エラー

|エラー|ベースエラー|説明|
|---|---|---|
|`ErrUnauthorizedInvalidToken`|`ErrUnauthenticated`|`Authenticator` によるトークン検証失敗|
|`ErrUnauthorizedTokenNotProvided`|`ErrUnauthenticated`|`Authorization` ヘッダーにトークンが見つからない|
|`ErrUnauthorizedTokenMissing`|`ErrUnauthenticated`|認証トークンが欠落|
|`ErrAuthnSlotNotFound`|`ErrUnauthenticated`|リクエストコンテキストに authn スロットが無い（`oapi.Middleware` が未注入）|
|`ErrInvalidAuthDefaultMode`|`ErrInternal`|デフォルト認証ポリシーが見つからない|

## authn スロット統合

この関数は OpenAPI バリデーションパイプライン内で動作するため、`echo.Context` ではなく `context.Context` のみが利用可能です。親の `oapi.Middleware` がバリデーション実行前に `request.Context()` へ **authn スロット**（`ctxhelper.WithAuthn`）を仕込むことで、バリデータから呼ばれる authFunc がそのスロットへ認証結果 `Authn` を `ctxhelper.SetAuthn` で書き戻せます。ハンドラは後段で `ctxhelper.GetAuthn` により取得します。

```mermaid
flowchart LR
    OapiMW["oapi.Middleware"] -->|"WithAuthn (seed slot)"| ReqCtx["request.Context()"]
    ReqCtx --> Validator["oapi validator → authFunc"]
    Validator -->|"SetAuthn"| ReqCtx
    Handler["handler"] -->|"GetAuthn"| ReqCtx
```

## 注意点

- トークン抽出はヘッダーのみ。Cookie は参照しません（Bearer / リソースサーバーモデル）
- Bearer プレフィックス除去は `AllowedHeaderBearer` が true かつヘッダー名が `Authorization` の場合のみ適用
- `Authenticator` の実装は環境固有（ローカルモック、JWT、OAuth 等）で DI 経由で注入される
