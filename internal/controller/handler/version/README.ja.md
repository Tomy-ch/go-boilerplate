# Version ハンドラ (`internal/controller/handler/version`)

[English](README.md) | 日本語

## 役割

`version` は、ビルド情報エンドポイント **`GET /version`** を公開します。

ビルド時に（`ldflags` で）バイナリへ埋め込まれたバージョンメタデータと、稼働中の
サービス識別情報を返します。これにより運用者やクライアントは、いま通信している
ビルドと実行環境を正確に判別できます。

liveness / readiness プローブとは異なり、これはヘルスチェックではありません。
静的なビルド / 識別情報を報告するだけで、データベースには一切アクセスしません。

## 標準ハンドラパターン

これは[親ハンドラガイド](../README.md)に記載された標準パターンに従う恒久ハンドラ
です。`server` 構造体を `BindHandler` が `gen.NewStrictHandler` /
`gen.RegisterHandlers` を通じて組み立て、ハンドラ本体を tracer span で包みます。

```go
func BindHandler(
    e *echo.Echo,
    tf observability.TracerFactory,
    loc *time.Location,
    bi system.BuildInfo,
    ac *config.ApplicationConfig,
)
```

- `bi system.BuildInfo` は、`ldflags` で注入された `Version()`・`Revision()`・
  `BuildDate()` の値を提供します。
- `ac *config.ApplicationConfig` は、稼働中の `Env()` と `Name()` を提供します。
- `loc *time.Location` は、ビルド日時の描画に用いる location です。
- `tf observability.TracerFactory` は、controller 層の `LayerTracer` を生成します。

`BindHandler` は controller DI モジュール
（[`internal/di/module/controller.go`](../../../di/module/controller.go)）で
`fx.Invoke(version.BindHandler)` として結線されています。

## レスポンス

`GetVersion` は `gen.VersionResponse`（`GetVersion200JSONResponse`）を返します。

| フィールド | 由来 |
| --- | --- |
| `Version` | `system.BuildInfo.Version()` |
| `Revision` | `system.BuildInfo.Revision()` |
| `BuildDate` | `system.BuildInfo.BuildDate()` を `loc` へ変換 |
| `Environment` | `config.ApplicationConfig.Env()` |
| `Service` | `config.ApplicationConfig.Name()` |

`BuildDate` は、RFC 3339 UTC 文字列を `datetime.ParseRFC3339UTCToLocation` で
注入された `*time.Location` へ変換して得ます。注入されたビルド日時が正しい
RFC 3339 UTC 値でない場合、ハンドラは `apperror.ErrInternal` をラップした
エラー（`invalid build date`）を返します。これはクライアント起因ではなく、
壊れたビルドを表します。

## 補足

このハンドラは下流の usecase 呼び出しを持ちません。再束縛した `ctx` を伝搬しない
ため、親ガイドのプローブ系ハンドラ向け例外に従い、
`_, endSpan := s.tracer.Start(ctx)` で span を開始します。
