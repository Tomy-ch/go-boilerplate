# logging

概要: `logging` ディレクトリは、アプリケーションのロギング機能を提供します。

このパッケージは、アプリケーション全体で一貫した構造化ロギングを提供します。`config.ApplicationConfig` に応じて開発用／本番用の設定を切り替えるファクトリ関数を持ち、テスト向けの簡易インスタンスも提供します。

主要な提供機能:

- **環境別ロガー生成**: `New`, `NewProductionLogger`, `NewDevelopmentLogger` により環境に合った `Logger` を返します。
- **テスト用ロガー**: `NewTestInstance(t *testing.T)` でテスト向けに `zap.NewNop()` ベースのロガーを生成します。
- **フィールドヘルパ**: `String`, `Int`, `Bool`, `Error`, `Any` などで安全にフィールドを作成できます。

## 役割

このディレクトリは次を実現します:

- 環境（開発 / 本番）に応じたデフォルト設定の提供
- アプリケーションコードから使いやすい `Logger` インターフェースの提供
- テストでの取り扱いを容易にする `NewTestInstance` の提供

## 必要度

### 本番運用での必須度

- **必須度: 本番運用で必須**
- 理由: 本番では構造化ログ（JSON）での出力が期待され、監視・障害解析・監査に利用されます。

### 開発/テスト運用での必須度

- **必須度: 開発/テスト運用で必須**
- 理由: 開発時は可読性の高いコンソール出力（色付き、詳細なデバッグレベル）を使うことで迅速なデバッグが可能です。テストでは `NewTestInstance` を使い副作用なしにロギングを扱えます。

## 利用例

### 使用例

```go
import (
  "boilerplate-go/internal/config"
  "boilerplate-go/internal/logging"
)

func main() {
  appCfg := config.Load() // アプリケーション設定取得（例）
  logger, err := logging.New(appCfg)
  if err != nil {
    panic(err)
  }

  logger.Info("server starting", logging.String("mode", appCfg.Mode()))
}
```

### フィールドの例

```go
logger.Error("failed to process", logging.String("user_id", "123"), logging.Error("err", err))
```

### テストでの利用例

```go
func TestSomething(t *testing.T) {
  logger := logging.NewTestInstance(t)
  // テスト内では zap の出力を行わない NOP ロガーが返る
  logger.Info("starting test")
}
```

テスト／モック:

- パッケージには `mock` ディレクトリに生成済みの mock があり、`Logger` インターフェースをモック化してユニットテストで注入できます（`go:generate mockgen` がトップに記載されています）。

運用上の推奨:

- 本番ではログを外部システムへ流す仕組み（ファイル、syslog、外部集約）を検討してください。
- 機密情報はログに平文で出力しないでください。必要に応じてマスク処理を行ってください。

### 無効化した場合の影響

- ロギングが不在だと、障害発生時の原因追跡や利用状況の把握が難しくなります。
- 特に本番でログを JSON 出力しないと外部ログ集約（例: ELK, CloudWatch）との連携が困難になります。

## 注意点

- 本番環境では JSON、開発環境では console 出力がデフォルトです。設定を変える場合は `zap.Config` を編集してください。
- `New` は `config.ApplicationConfig` の `Mode()` を利用して生成するロガーを切り替えます。アプリケーションの設定が正しく初期化されていることを確認してください。
- ログフィールドは必ず `String`, `Int` などのヘルパを使って作成してください。`ConvertFields` により内部で `zap.Field` に変換されます。
- パフォーマンスを気にする場合、低コストで済むようにログレベルのチェック（本体の `zap` が高速に処理）や `Named`/`CallerSkip` の使い分けを検討してください。
