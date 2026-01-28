# auth 境界インターフェース

概要: `internal/usecase/boundary/auth` パッケージは、認証に関する入力/出力の境界（DTO とインターフェース）を定義します。アクセストークン等の `Credential`、認証結果を表す `Authn`、および認証を行う `Authenticator` インターフェースとエラー定義を提供します。

## 役割

- 認証情報の表現 (`Credential`) を提供する。
- 認証結果の表現 (`Authn`) を提供する。subject の UUID 解析やスコープ/クレーム保持を行う。
- 認証処理の境界を表す `Authenticator` インターフェースを定義する（DI により実装を差し替える想定）。
- 共通のエラー値（トークン欠如・未認証・ID 未存在など）を定義し、上位層で統一的に扱えるようにする。

## 実装の要点

- `Credential`:
  - `NewCredential(accessToken string)` で生成。空トークンはエラー `ErrArgumentTokenMissing`。
  - `AccessToken()` でトークン取得。
- `Authn`:
  - `New(subject, provider, scopes, claims)` で生成。空の subject は `ErrUnauthorizedSubjectMissing`。
  - subject が UUID 形式なら内部でパースして `ID()` で取得可能（`HasID()` で存在チェック）。UUID でない場合は `ID()` は `ErrInvalidIDMissing` を返す。
  - `Provider()`, `Scopes()`, `Claims()` を通じてメタ情報を取得できる。
- `Authenticator` インターフェース:
  - `Authenticate(ctx context.Context, cred *Credential) (*Authn, error)` を定義。実装は外部プロバイダ呼び出しやトークン検証ロジックを担う。
- エラー定義:
  - `ErrUnauthorizedSubjectMissing`, `ErrInvalidIDMissing`, `ErrArgumentTokenMissing` を提供し、アプリケーション層で一貫したエラー処理を行えるようにしている。

## 使い方

- トークンから認証情報を作る例:

```go
cred, err := auth.NewCredential(accessToken)
if err != nil { /* handle */ }
authn, err := authenticator.Authenticate(ctx, cred)
if err != nil { /* handle unauthenticated */ }
ctxhelper.SetAuthnToEcho(ec, *authn) // Echo コンテキストにセット
```

## 前提 / 要件

- `Authenticator` の実装は、authenticator.AuthenticateのIFを満たすことで認証ロジックを提供できます。例えば、OAuth2 トークン検証や JWT 検証など、利用する認証方式に応じた実装をDIで差し替え可能です。
- `Authn.New` は subject のトリミング・簡易検証のみを行うため、必要に応じて呼び出し元で追加の検証や監査を行ってください。

## 注意点

- `Authn` は subject をそのまま保持しつつ、UUID として解釈できれば `ID()` を提供します。UUID を前提とする処理では `HasID()` を確認するか `ID()` のエラーを扱ってください。
- エラーはアプリケーションの `apperror` を元にラップされた値が返り、errorhandlerで適切に HTTP ステータスコードに変換されます。そのため、errorhandlerのミドルウェアは先に登録してください。
