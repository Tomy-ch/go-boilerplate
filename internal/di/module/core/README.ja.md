# core モジュール

[English](README.md) | 日本語

`internal/di/module/core` は、HTTP スタックで共通利用される **コアコンポーネントの DI モジュール群**を提供するパッケージです。

各モジュールは `fx.Option` を返し、対応するコンポーネントを DI コンテナに登録します。

## モジュール一覧

|関数|ファイル|提供するコンポーネント|
|---|---|---|
|`AuthnModule()`|`auth.go`|認証（Authenticator + Auth コントローラ）|
|`BasicAuthModule()`|`basicauth.go`|メトリクスエンドポイント用 Basic 認証バリデータ|
|`SecurityCookieModule()`|`security_cookie.go`|Cookie セキュリティ属性の設定|
|`SkipperModule()`|`skipper.go`|OpenAPI バリデーションの ops エンドポイントスキップ|
|`ValidatorModule()`|`validator.go`|OpenAPI スキーマバリデータ|

## 設計方針

- 各モジュールは1ファイル = 1モジュールで分離
- 内部実装は `internal/controller/httpstack` 等のコンストラクタを `fx.Provide` でラップするだけ
- ビジネスロジックを含まない

## Test Strategy

ここに置かれるモジュールはいずれも `internal/controller/httpstack` 等の純粋なコンストラクタをラップするだけで、起動に実インフラを必要とするものはクロージャに含まれません。そのため上位層では負担できない 2 段目の検証が可能で、各モジュールは 2 本の sibling テストを持ちます。

- **`Test<Module>_GraphIsValid`** — [`../README.md`](../README.ja.md) が宣言する層のベースライン。`fx.ValidateApp` がモジュールと宣言された依存をまとめて解決し、型の欠落なくグラフが結線されていることを、コンストラクタやライフサイクルフックを **実行せずに** 検証します。
- **`Test<Module>`** — 最小の `fx.New` アプリを起動し、`fx.Populate` した提供コンポーネントの値そのものを assert します（例: 環境ゲートが `local.New()` を選んだこと）。グラフ検証では到達できない「コンストラクタが実際に走り、使えるコンポーネントを生成すること」を担保します。

2 段目が成立するのは、**ここのモジュールが起動に実インフラを必要としないから**です。基準はディレクトリではなくこの性質にあります。クロージャが DB やネットワーク接続を要求するモジュールは上位に置き、そこでは `fx.ValidateApp` だけが戦略になります。

固有のロジックを持つ provider 本体（`provideAuthenticator` / `provideJWKSAuthenticator`）は、[`../../README.md`](../../README.ja.md) の DI 層ベースラインに従って直接ユニットテストします。グラフ検証はどちらにも到達せず、環境ゲートの拒否ケースこそが要点だからです。

## 注意点

- モジュールの追加・削除は `internal/di/module` の上位モジュールから参照を変更する必要がある
