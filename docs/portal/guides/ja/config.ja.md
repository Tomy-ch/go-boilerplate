# `Config` パッケージ

アプリケーション全体で利用するコンフィグファイルを定義しておくディレクトリです。

## 実装について

このパッケージは、環境変数からアプリケーション設定を読み取り、型付きの構造体としてアプリケーション全体に提供する責務を持ちます。主要な実装の流れは次のとおりです。

- `Loader`（テスト用の `Loader` を含む）に環境変数をパースし、`env.ParseAs[Loader]()` によって値を読み込みます（実装ファイル: `loader.go` / `envspec.go`）。
- 読み込んだ `Loader` の値を `Config` 型に変換し、内部で必要に応じたバリデーション（`validateConfig()`）を行います（実装ファイル: `config.go`, `model.go`）。
- 生成された `*Config` は不変（setter を公開しない）として扱い、DI コンテナに登録するためのセットアップ関数（`SetUpConfig`）を通じてアプリケーションに注入されます（DI 関連: `internal/di/module/config.go`）。

設計上のポイント:

- 設定は初期化時にまとめて読み取り、ランタイム中に値を変更しない（イミュータブルに扱う）。
- 必須値の欠落は `validateConfig()` で検出し、起動失敗（明示的なエラー返却）させることで不正な状態で稼働しないようにしています。
- テスト用のヘルパーやモック（`config_testing_mock.go`, `config_testing_setter.go`）を用意しており、テスト環境で環境変数を差し替えて `New()` の挙動を検証できます。

## Config Loading フロー

アプリケーション起動時の設定読み込みの流れは次の通りです。

```txt
.env files
    ↓
Load (godotenv)
    ↓
env.ParseAs[Loader]()
    ↓
validateConfig()
    ↓
Config struct
    ↓
SubConfig Provider<br/>(NewServerConfig など)
    ↓
DI (Uber Fx)
```

### 各ステップの役割

- **Load()**
  - `.env/.env` および `.env/.env.<ENV>` を読み込み、環境変数をセットします。

- **env.ParseAs[Loader]**
  - `envspec.go` に定義された `Loader` 構造体へ環境変数をマッピングします。

- **validateConfig()**
  - ポート範囲、CIDR、タイムアウト値などの妥当性を検証します。
  - 不正な設定がある場合は起動時にエラーを返します。

- **Config struct**
  - `Loader` の値を内部構造体 (`model.go`) に変換します。
  - 外部から直接変更できない **immutable な設定オブジェクト**として扱います。

- **SubConfig Provider**
  - `NewServerConfig` や `NewDatabaseConfig` などの関数を通じて
  - 必要な設定のみを各コンポーネントに注入します。

- **DI (Uber Fx)**
  - `internal/di/module/config.go` で `fx.Provide` に登録され
  - アプリケーション全体で利用されます。

## 設計原則

このConfigパッケージは、次の設計原則に基づいて実装されています。

- **Configuration is immutable**
  - 設定は起動時に一度だけ読み込まれ、ランタイム中に変更されません。

- **Configuration is loaded only at startup**
  - `.env` → `Loader` → `validateConfig()` → `Config` の順に初期化されます。

- **Domain / Usecase は環境変数に依存しない**
  - 環境変数の解釈は Config 層に閉じ込めます。

- **Typed SubConfig を通じて設定を取得する**
  - `NewServerConfig` や `NewDatabaseConfig` などの provider を通じて、
    必要な設定のみを各コンポーネントに渡します。

これにより、アプリケーションの設定依存を最小限にし、テスト容易性と保守性を高めています。

## メインで利用しているライブラリ

- `github.com/caarlos0/env/v11` — 環境変数を構造体へマッピングするために使用しています（タグベースで `env:"KEY,required"` を指定可能）。
- DI 登録には Uber Fx のパターンを採用しているため、設定値は `fx.Provide` を通じて注入されます（DI 実装: `internal/di/module/config.go`）。
- テストでは `testify/require` 等の断言ライブラリと、リポジトリ内の `config_testing_*` ヘルパーを併用して、環境変数ベースの振る舞いを確実に検証しています。

## 必要度

以下は現実運用での重要度と期待される影響です。

### 本番運用での必須度

- 必須度: 高
  - 理由: データベース接続情報、外部サービスの認証情報、リスニングポート、CORS 設定、セキュリティ設定（HSTS 等）など、稼働に必須な値が環境変数で提供されるため、正確に読み取れないと起動失敗や重大な障害につながります。
  - 対策: 必須フィールドは `validateConfig()` で起動前に検証し、欠落時は明示的にエラーを返す設計にしてください。

### 開発/テスト運用での必須度

- 必須度: 高（ただしデフォルトやモックで代替可能）
  - 理由: テストでは特定の ENV をセットして挙動を検証するため、環境変数の管理が重要です。リポジトリ内にテスト用ヘルパーがあるため、CI やローカルテストでは `t.Setenv` と組み合わせて再現可能にします。
  - 例: `config_testing_setter.go` を使ってテスト用の設定を注入し、`New()` が正しくバインドすることを確認します。

### 無効化した場合の影響

- 影響の範囲: 軽微〜致命的（設定項目による）
  - 軽微: ログレベルやデバッグフラグなど、一部のオプションはデフォルト値で安全に代替できます。
  - 致命的: DB 接続情報や外部 API キーが欠落すると起動失敗やランタイムエラーを引き起こします。
  - 推奨対応: 本番での運用前に `go build` と `go test ./...` を実行し、`validateConfig()` によるチェックが通ることを確認してください。CI パイプラインで必須 ENV を検証する習慣を付けると安全です。

### 新たなカテゴリ(例: AWSやGCPなど)を追加する方法

1. `model.go`に新しいカテゴリ用の公開構造体（例: `AWS`や`GCP`など）を追加してください。  
    - 例:

    ```go
    type aws struct {
        accessKey string
        secretKey string
        region    string
    }
    ```

2. 追加したカテゴリの構造体を `Config` 構造体にフィールドとして追加してください。
    - 例:

    ```go
    type Config struct {
        server server
        aws    aws // 新しいカテゴリの追加
    }
    ```

3. `envspec.go` に新しいカテゴリの構造体を定義してください。
    - 各フィールドには `env` タグを付与し、必要に応じて `required` や `envSeparator` なども指定してください。
    - `model.go` で定義した構造体の公開構造体を `envspec.go` にも追加してください。
    - 例:

    ```go
    type AWS struct {
        AccessKey string `env:"AWS_ACCESS_KEY,required"`
        SecretKey string `env:"AWS_SECRET_KEY,required"`
        Region    string `env:"AWS_REGION,required"`
    }
    ```

4. 追加したカテゴリの構造体を `Loader` 構造体にフィールドとして追加してください。
    - 例:

    ```go
    type Loader struct {
        Server Server
        AWS    AWS // 新しいカテゴリの追加
    }
    ```

5. `config.go` の `New` 関数で `env.ParseAs[Loader]()` を呼び出すことで、環境変数から`Loader`に自動的に値がバインドされます。
    - 必要に応じて、追加カテゴリのバリデーション処理を `validateConfig()` 関数内で実装してください。
    - 最後に、`Loader` から `Config` への変換を行い、`Config` 構造体を返すようにしてください。

    ```go
    func New() (*Config, error) {
        cfg, err := env.ParseAs[Loader]()
        
        if err := validateConfig(cfg); err != nil { // バリデーション処理を追加はこの関数内で行います
            return nil, err
        }

        return &Config{
            server: server{
                host:           cfg.Server.Host,
                port:           cfg.Server.Port,
                allowedOrigins: cfg.Server.AllowedOrigins,
            },
            aws: aws{
                accessKey: cfg.AWS.AccessKey,
                secretKey: cfg.AWS.SecretKey,
                region:    cfg.AWS.Region,
            },
    }, nil
    }
    ```

6. 必要に応じて、カテゴリごとのgetterメソッドを実装し、外部から値を取得できるようにしてください。
    - setterメソッドの作成は禁止です。config.goの目的は、環境変数からの値の取得とバインドに限定されているため、設定値を変更することは想定されていません。
    - 例:

    ```go
    func (c *Config) AWSAccessKey() string {
        return c.aws.accessKey
    }

    func (c *Config) AWSSecretKey() string {
        return c.aws.secretKey
    }

    func (c *Config) AWSRegion() string {
        return c.aws.region
    }
    ```

7. `config_test.go` に追加したカテゴリのテストケースを追加してください。
    - テストケースでは、環境変数を設定し、`New()` 関数を呼び出して値が正しくバインドされていることを確認します。
    - 例:

    ```go
    func TestNewAWSConfig(t *testing.T) {
        expectedAccessKey := "test_access_key"
        expectedSecretKey := "test_secret_key"
        expectedRegion := "us-west-2"

        t.Setenv("AWS_ACCESS_KEY", expectedAccessKey)
        t.Setenv("AWS_SECRET_KEY", expectedSecretKey)
        t.Setenv("AWS_REGION", expectedRegion)

        cfg, err := New()
        require.NoError(t, err)

        require.Equal(t, expectedAccessKey, cfg.aws.accessKey)
        require.Equal(t, expectedSecretKey, cfg.aws.secretKey)
        require.Equal(t, expectedRegion, cfg.aws.region)
    }
    ```

    - validateConfig関数内でのバリデーションのテストも追加してください。

8. 既存の `NewHogehogeConfig` と同様に、DI が `*config.Config` を受け取り、公開型（`*config.AWSConfig`）を返すシグネチャにしてください。

```go
// internal/config/aws_config.go (例)
package config

type AWSConfig struct {
    AccessKey string
    SecretKey string
    Region    string
}

// NewAWSConfig は DI から渡された *config.Config を受け取り、
// サービスで使う公開型 *config.AWSConfig を返します。
func NewAWSConfig(cfg *Config) *AWSConfig {
    return &AWSConfig{
        AccessKey: cfg.aws.accessKey,
        SecretKey: cfg.aws.secretKey,
        Region:    cfg.aws.region,
    }
}
```

ポイント:

- フィールド名や構造体名はプロジェクトの命名規則に従ってください。
- DI に provider を追加したら、その型を受け取るコンポーネント（例: AWS クライアントの factory）を DI コンテナに登録してください。
- 追加漏れや型の不一致があるとビルド時にエラーになるため、`go build` や `go test ./...` で確認してください。

## 注意点

- セキュリティ上の理由から、機密情報（例: APIキーやパスワード）は環境変数で管理し、コード内にハードコーディングしないでください。
