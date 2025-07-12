# Config

アプリケーション全体で利用するコンフィグファイルを定義しておくディレクトリです。

## 実装について

## メインで利用しているライブラリ

- github.com/caarlos0/env/v11
  - 環境変数を構造体にバインドするためのライブラリ

### 新たなカテゴリ(例: AWSやGCPなど)を追加する方法

1. `types.go`に新しいカテゴリ用の公開構造体（例: `AWS`や`GCP`など）を追加してください。  
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

3. `private.go` に新しいカテゴリの構造体を定義してください。
    - 各フィールドには `env` タグを付与し、必要に応じて `required` や `envSeparator` なども指定してください。
    - `types.go` で定義した構造体の公開構造体を `private.go` にも追加してください。
    - 例:

    ```go
    type AWS struct {
        AccessKey string `env:"AWS_ACCESS_KEY,required"`
        SecretKey string `env:"AWS_SECRET_KEY,required"`
        Region    string `env:"AWS_REGION,required"`
    }
    ```

4. 追加したカテゴリの構造体を `ConfigLoader` 構造体にフィールドとして追加してください。
    - 例:

    ```go
    type ConfigLoader struct {
        Server Server
        AWS    AWS // 新しいカテゴリの追加
    }
    ```

5. `config.go` の `New` 関数で `env.ParseAs[ConfigLoader]()` を呼び出すことで、環境変数から`ConfigLoader`に自動的に値がバインドされます。
    - 必要に応じて、追加カテゴリのバリデーション処理を `validateConfig()` 関数内で実装してください。
    - 最後に、`ConfigLoader` から `Config` への変換を行い、`Config` 構造体を返すようにしてください。

    ```go
    func New() (*Config, error) {
        cfg, err := env.ParseAs[ConfigLoader]()
        
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

        assert.Equal(t, expectedAccessKey, cfg.aws.accessKey)
        assert.Equal(t, expectedSecretKey, cfg.aws.secretKey)
        assert.Equal(t, expectedRegion, cfg.aws.region)
    }
    ```

    - validateConfig関数内でのバリデーションのテストも追加してください。

> ※ フィールド名や構造体名はプロジェクトの命名規則に従ってください。
