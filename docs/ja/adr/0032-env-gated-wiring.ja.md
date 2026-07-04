---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [di, config]
---

# ADR-0032: DI を通じて環境ごとに実装を切り替える（環境ゲート結線）

English canonical: [0032-env-gated-wiring.md](../../adr/0032-env-gated-wiring.md)

## ステータス

accepted

## 背景

一部のコンポーネントはデプロイ環境によって異なる具体的な実装を必要とする。認可処理がその典型例である。本番環境には完全な RBAC または外部ポリシーエンジンの実装が必要だが、ローカル開発・CI・テスト環境は外部ポリシーインフラに依存しないシンプルなスタブで十分だ。

アプリケーションコード（ドメイン・ユースケース・コントローラー）内で環境分岐が発生すると、プロトコルや環境の知識が内側のレイヤーに漏れ込む。暗黙的な分岐（フィーチャーフラグや遅延バインディング）では起動時の保証が弱まり、誤った実装が本番環境でサイレントに動作するリスクがある。

## 決定

具体的な実装の環境分岐は、コンポーネントを構築する DI モジュールプロバイダー内、すなわち**コンポジションルート**でのみ行う。プロバイダーは `config.ApplicationConfig` から環境識別子を読み取り、適切な実装を選択して、宣言されたインターフェース型として返す。

本番非互換のスタブは**フェイルクローズ**する。現在の環境に対して本番実装が結線されていない場合、プロバイダーはエラーを返して `fx.App` の起動を失敗させ、安全でないスタブがサイレントに動作することを防ぐ。非本番スタブが結線された場合は `WARN` レベルのログを出力して代替を可視化する。

認可プロバイダー（`internal/di/module/authz.go` の `provideAuthorizer`）がこのパターンの例である。

```go
switch appCfg.Env() {
case config.EnvLocal, config.EnvCI, config.EnvTest:
    logger.Warn("Allow-all authorizer wired: every request is permitted (non-production only)", ...)
    return allowall.New(), nil
default:
    logger.Error("No authorizer configured for the current environment", ...)
    return nil, xerrors.New("no authorizer configured for environment: " + appCfg.Env())
}
```

## 影響

### ポジティブな影響

- コンポジションルートが環境固有の実装選択を行う唯一の監査可能な場所になる。
- 安全でないスタブが本番環境にサイレントに到達できない。プロバイダーは起動エラーとログエントリでフェイルクローズする。
- 内側のレイヤー（ユースケース・ドメイン）は宣言されたインターフェースにのみ依存し、どの具体的な実装が注入されているかを意識しない。

### ネガティブな影響

- 各環境ゲートプロバイダーはすべての分岐（allow-all パスとフェイルクローズパス）のテストカバレッジが必要である。そうでなければ選択ロジックはグラフ検証テストによってのみ検証される。
- 新しい環境バリアントが導入された場合、新しいプロバイダーを作成するか既存のものを拡張する必要がある。

## 検討した代替案

### ランタイムのフィーチャーフラグ

フラグまたは設定値を通じて実装の選択をランタイムに先送りする。却下: 起動時に推論しにくく、設定ミスのあるフラグが起動失敗なしに本番で誤った実装をサイレントに実行する可能性がある。

### 環境ごとに別の DI モジュールファイル

環境ごとに個別のモジュールファイルをコンパイルする。却下: ビルドの複雑さが増し、シングルバイナリモデルから乖離する。

## 補足

- 出典: `internal/di/module/authz.go`（`provideAuthorizer`、29–47 行）、`internal/di/module/README.md`。
- モジュール README はこのパターンを「環境ゲート: ローカル / CI / テストにのみ allow-all スタブを結線し、それ以外ではフェイルクローズ（エラーを返す）する」と記載している。
- 設定定数（`EnvLocal`、`EnvCI`、`EnvTest`）は `internal/config` で定義されている。
