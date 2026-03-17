# `Config` Package

This directory defines configuration used across the entire application.

## Implementation

This package is responsible for reading application settings from environment variables and providing them as **typed configuration structures** throughout the application.

The main implementation flow is as follows:

- Environment variables are parsed into a `Loader` structure (including a testing `Loader`) using `env.ParseAs[Loader]()` (implementation files: `loader.go` / `envspec.go`).
- The loaded `Loader` values are converted into the `Config` type, and internal validation (`validateConfig()`) is executed when necessary (implementation files: `config.go`, `model.go`).
- The generated `*Config` is treated as immutable (no setters are exposed) and injected into the application through a setup function (`SetUpConfig`) registered in the DI container (DI reference: `internal/di/module/config.go`).

Design highlights:

- Configuration is loaded at startup and never modified during runtime (treated as immutable).
- Missing required values are detected by `validateConfig()`, causing the application to fail fast during startup instead of running in an invalid state.
- Testing helpers and mocks (`config_testing_mock.go`, `config_testing_setter.go`) are provided so environment variables can be overridden in test environments to verify `New()` behavior.

## Config Loading Flow

The configuration loading process during application startup is shown below.

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
SubConfig Provider<br/>(NewServerConfig etc.)
    ↓
DI (Uber Fx)
```

### Role of Each Step

- **Load()**
  - Loads `.env/.env` and `.env/.env.<ENV>` and sets environment variables.

- **env.ParseAs[Loader]**
  - Maps environment variables into the `Loader` struct defined in `envspec.go`.

- **validateConfig()**
  - Validates values such as port ranges, CIDR blocks, and timeout values.
  - If an invalid configuration exists, startup fails with an explicit error.

- **Config struct**
  - Converts `Loader` values into internal structures (`model.go`).
  - Treated as an **immutable configuration object** that cannot be modified externally.

- **SubConfig Provider**
  - Functions such as `NewServerConfig` and `NewDatabaseConfig`
  - Inject only the required configuration values into each component.

- **DI (Uber Fx)**
  - Registered through `fx.Provide` in `internal/di/module/config.go`
  - Used across the entire application.

## Design Principles

This Config package is implemented based on the following principles.

- **Configuration is immutable**
  - Configuration is loaded only once during startup and never modified at runtime.

- **Configuration is loaded only at startup**
  - Initialization sequence: `.env` → `Loader` → `validateConfig()` → `Config`.

- **Domain / Usecase must not depend on environment variables**
  - Interpretation of environment variables is isolated within the Config layer.

- **Configuration should be accessed through typed SubConfig providers**
  - Providers such as `NewServerConfig` or `NewDatabaseConfig` expose only the required configuration for each component.

This design minimizes configuration coupling and improves testability and maintainability.

## Main Libraries Used

- `github.com/caarlos0/env/v11`  
  Used to map environment variables into Go structs (tag-based configuration such as `env:"KEY,required"`).

- Uber Fx DI pattern  
  Configuration values are injected through `fx.Provide` (DI implementation: `internal/di/module/config.go`).

- Testing libraries  
  `testify/require` together with `config_testing_*` helpers are used to reliably verify environment-variable-based behavior.

## Importance

The following describes the operational importance and expected impact.

### Importance in Production

- Importance: **High**

Reason:

Database connection information, external service credentials, listening ports, CORS configuration, and security settings (HSTS etc.) are provided through environment variables. If these cannot be loaded correctly, the application may fail to start or encounter serious runtime failures.

Recommended mitigation:

Required fields should be validated by `validateConfig()` before startup, and missing values should cause explicit startup errors.

### Importance in Development / Testing

- Importance: **High (but replaceable with defaults or mocks)**

Reason:

Tests rely on specific environment variables to validate behavior. Proper environment management is therefore important.

Because test helpers exist in the repository, CI and local tests can reproduce behavior using `t.Setenv`.

Example:

Use `config_testing_setter.go` to inject test configuration values and verify that `New()` binds them correctly.

### Impact if Disabled

Impact range: **Minor to Critical (depending on configuration)**

- Minor: Some options such as log levels or debug flags may fall back to safe defaults.
- Critical: Missing DB connection info or external API keys will cause startup failures or runtime errors.

Recommended practice:

Before production deployment, run:

```bash
go build
go test ./...
```

Ensure `validateConfig()` passes.  
It is also recommended to validate required environment variables in CI pipelines.

### How to Add a New Category (Example: AWS / GCP)

1. Add a new private structure for the category in `model.go`.

    ```go
    type aws struct {
        accessKey string
        secretKey string
        region    string
    }
    ```

2. Add the new category as a field in the `Config` struct.

    ```go
    type Config struct {
        server server
        aws    aws // newly added category
    }
    ```

3. Define the corresponding structure in `envspec.go`.

    Each field must include an `env` tag.

    ```go
    type AWS struct {
        AccessKey string `env:"AWS_ACCESS_KEY,required"`
        SecretKey string `env:"AWS_SECRET_KEY,required"`
        Region    string `env:"AWS_REGION,required"`
    }
    ```

4. Add the category to the `Loader` struct.

    ```go
    type Loader struct {
        Server Server
        AWS    AWS // newly added category
    }
    ```

5. `env.ParseAs[Loader]()` automatically binds environment variables to the `Loader`.

    Add validation if needed in `validateConfig()` and convert `Loader` into `Config`.

    ```go
    func New() (*Config, error) {
        cfg, err := env.ParseAs[Loader]()

        if err := validateConfig(cfg); err != nil {
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

6. Implement getter methods if external access is required.

    Setter methods **must not be implemented**.

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

7. Add test cases in `config_test.go`.

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

    Also add validation tests for `validateConfig()`.

8. Provide a public SubConfig provider similar to existing ones.

    ```go
    // internal/config/aws_config.go (example)
    package config

    type AWSConfig struct {
        AccessKey string
        SecretKey string
        Region    string
    }

    // NewAWSConfig receives *config.Config from DI
    // and returns the public type*config.AWSConfig used by services.
    func NewAWSConfig(cfg *Config)*AWSConfig {
        return &AWSConfig{
            AccessKey: cfg.aws.accessKey,
            SecretKey: cfg.aws.secretKey,
            Region:    cfg.aws.region,
        }
    }
    ```

Notes:

- Follow project naming conventions for struct and field names.
- After adding a provider to DI, register components that consume that type (e.g., an AWS client factory).
- Missing registrations or type mismatches will cause build errors. Always verify with:

```bash
go build
go test ./...
```

## Notes

For security reasons, sensitive information (such as API keys or passwords) must be managed through environment variables and **must not be hardcoded in source code**.
