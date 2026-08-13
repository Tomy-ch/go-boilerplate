# buildinfo

[English](README.md) | 日本語

`internal/observability/metrics/buildinfo` は、**アプリケーションのビルド・バージョン・ランタイム情報を Prometheus の info gauge (`app_build_info`) として公開する**パッケージです。

`/version` HTTP エンドポイントを補完し、稼働中のバージョンをメトリクスバックエンド (Prometheus / Grafana) から横断的に確認できるようにします。デプロイ確認・障害調査・ロールバック判断に役立ちます。

## 公開 API

|関数 / 型|説明|
|---|---|
|`Collector`|`prometheus.Collector` を実装するビルド情報コレクター|
|`NewCollector(appCfg, bi)`|全ラベル値を結線時に一度だけ解決・正規化して `Collector` を生成|
|`Register(c)`|コレクターを Prometheus のデフォルトレジストリに登録（重複登録は無視）|

## メトリクス

|項目|値|
|---|---|
|名前|`app_build_info`|
|型|Gauge|
|値|常に `1`（意味はラベルに持たせる）|

### ラベル

|ラベル|例|取得元|
|---|---|---|
|`service`|`go-boilerplate`|`config.ApplicationConfig.Name()`|
|`environment`|`production`|`config.ApplicationConfig.Env()`|
|`version`|`v1.5.0`|`system.BuildInfo.Version()`|
|`revision`|`abcdef1`|`system.BuildInfo.Revision()`|
|`build_date`|`2026-06-28T17:00:00Z`|`system.BuildInfo.BuildDate()`|
|`go_version`|`go1.24.4`|`runtime.Version()`|

## 設計方針

- **`/version` と同一の source of truth**: ビルド値は `system.BuildInfo` から取得するため、`/version` と `app_build_info` が異なる値を持つことはありません。
- **結線時に値を実体化**: ビルド・バージョン・ランタイム情報は起動後に変化しないため、全ラベル値を `NewCollector` で**一度だけ**解決・正規化して保持します。`Collect` は保持済みの値を `MustNewConstMetric` で emit するだけで、スクレイプごとの計算は行いません（可変な `pgxpool` プール統計コレクターとは異なる方針です）。
- **空値は `unknown` に丸める**: 空のラベル値は結線時に `unknown` へ正規化します。

## 注意点

- 高カーディナリティ化や秘匿・環境情報の漏えいを避けるため、次のラベルは意図的に**含めません**: `hostname` / `pod_name` / `container_id` / `instance_id` / `git_branch` / `build_url` / `image_digest` / `full_image` / `token` / `registry` / `commit`。
- 本パッケージは `os.Getenv` を直接呼びません。環境値は不変な `config.ApplicationConfig` から取得します。
- ビルド情報を domain / usecase 層に持ち込みません。
- `prometheus.AlreadyRegisteredError` を無視して重複登録を安全にスキップします。
