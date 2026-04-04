# SecureCookie 設定 README

このパッケージは、レスポンスの `Set-Cookie` を **安全な属性へ正規化（rewrite）**するための仕組みです。  
アプリ側の各ハンドラ／ユースケースが都度 `Secure/HttpOnly/SameSite/Path/Domain` を意識しなくても、**ミドルウェア層で一括適用**できるのが狙いです。

## 何をするか

- `Set-Cookie` の属性を、`SecurityCookie` の設定に従って **付与/上書き/削除**します
- `__Host-` / `__Secure-` などの **Cookie Prefix ルール**にも従い、危険な指定を落とします
- 失敗時（解析できないヘッダなど）は、安全のため `RewriteSetCookie` が空文字を返します  
  → 呼び出し側（ResponseWriter wrapper）は **元の raw を通す**運用が推奨です

## NewSecurityCookie の方針（推奨）

`NewSecurityCookie(p *config.SecureCookieConfig)` は、基本的に以下の思想で初期化します。

- `applyToAll = true`（原則すべての `Set-Cookie` を対象にする）
- `forceHTTPOnly = true`（原則 HttpOnly を付ける）
- `enforceSecureWhenSameSiteNone = true`（SameSite=None の場合 Secure を強制）
- `forcePath = "/"`（多くのケースで path は `/` 固定で安全）

一方で、`Secure` / `SameSite` / `Domain` は `SecureCookieConfig` から調整可能にしてあります。

## 設定値一覧と意味

### applyToAll

- **意味**: `Set-Cookie` を **全部 rewrite 対象**にするかどうか
- **推奨**: `true`
- **補足**: `false` にすると `cookieNames` で指定した Cookie 名のみ rewrite します

### cookieNames

- **意味**: `applyToAll=false` の場合に、rewrite 対象とする Cookie 名のホワイトリスト
- **型**: `map[string]struct{}`
- **推奨**: 基本は使わず `applyToAll=true` のままが楽
- **ユースケース**: “フレームワークが勝手に吐く Set-Cookie には手を入れたくない” など

### skipCookieNames

- **意味**: rewrite 対象から除外する Cookie 名（ブラックリスト）
- **型**: `map[string]struct{}`
- **推奨**: 例外が出たときだけ追加
- **ユースケース**: 特定のCookieだけ別ポリシーで運用したい / 互換性のため触りたくない

### forceSecure

- **意味**: `Secure` 属性を強制する（`nil` なら上書きしない）
- **型**: `*bool`
- **挙動**:
  - `ptr.To(true)`  → 常に `Secure` を付与
  - `ptr.To(false)` → 常に `Secure` を削除（※基本非推奨）
  - `nil`           → 既存値を尊重（ただし prefix の強制は別で入る）
- **推奨**:
  - 本番/ステージング: `true`
  - ローカルHTTP: `nil` or `false`（ただし SameSite=None を使うなら破綻しやすい）

### forceHTTPOnly

- **意味**: `HttpOnly` 属性を強制する（`nil` なら上書きしない）
- **型**: `*bool`
- **推奨**: `true`
- **注意**: JS から Cookie を読む設計（SPAで `document.cookie` を読む等）だと `HttpOnly` で読めません  
  → その設計自体を避け、Authorization Header や BFF を推奨

### forceSameSite

- **意味**: `SameSite` を強制上書きする（空文字なら上書きしない）
- **型**: `string`
- **許容値**: `"Lax" / "Strict" / "None" / ""`
- **推奨**:
  - まずは `"Strict"` または `"Lax"`
  - クロスサイト用途（別ドメインPOSTなど）がある場合のみ `"None"` を検討

### enforceSecureWhenSameSiteNone

- **意味**: `SameSite=None` を使う場合に `Secure` を強制付与する
- **型**: `bool`
- **推奨**: `true`
- **理由**: ブラウザ仕様として `SameSite=None` は `Secure` が必須に近い（安全側）

### forcePath

- **意味**: `Path` を強制上書きする（空文字なら上書きしない）
- **型**: `string`
- **推奨**: `"/"`
- **補足**:
  - `__Host-` の場合は仕様上 `Path=/` が必須なので、最終的に必ず `/` になります

### forceDomain

- **意味**: `Domain` を強制上書きする（空文字なら上書きしない）
- **型**: `string`
- **推奨**: 基本は空（未指定）
- **注意**:
  - `__Host-` の場合は **Domain属性禁止**なので `RewriteSetCookie` 内で削除されます
- **ユースケース**:
  - サブドメイン間で共有したい（例: `.example.com`）など  
    ※セキュリティ/運用事故が増えるので慎重に

### forceMaxAge

- **意味**: `Max-Age` を強制上書きする（`nil` なら上書きしない）
- **型**: `*int`
- **推奨**: 通常 `nil`（Cookie個別に設定）
- **ユースケース**:
  - “全Cookieの有効期限を短く強制したい” など（ただし副作用が大きい）

## Cookie Prefix の扱い（自動ルール）

この実装は Cookie 名の prefix に応じて **追加の強制**を行います。

### `__Secure-`

- 強制されること:
  - `Secure` を必ず付与

### `__Host-`

- 強制されること:
  - `Secure` を必ず付与
  - `Path=/` を必ず付与
  - `Domain` を必ず削除

このため、入力がこうでも…

```text
Set-Cookie: __Host-access_token=rawtoken; Path=/hoge; Domain=example.com; SameSite=None
```

最終的にこうなります（例）:

```text
Set-Cookie: __Host-access_token=rawtoken; Path=/; SameSite=Strict; Secure; HttpOnly
```

## config.SecureCookieConfig から反映される値

`NewSecurityCookie()` は以下を `SecureCookieConfig` から取り込みます。

- `cfg.forceSecure = p.Secure()`
- `cfg.forceSameSite = p.SameSite()`（空なら上書きしない）
- `cfg.forceDomain = p.Domain()`（空なら上書きしない）

つまり **環境ごとに**（例: local/staging/prod）ポリシーを切り替えたい場合は、`SecureCookieConfig` を config 層で分岐させるのが自然です。

## 注意点

- `SameSite=None` を使うときは `Secure=true` がほぼ必須です  
  （この実装は `enforceSecureWhenSameSiteNone=true` で安全側に倒します）
- `Domain` を広げるほど Cookie が届く範囲が広がり、事故りやすくなります
- `HttpOnly=false` にすると XSS 影響が大きくなります。まず避けてください
