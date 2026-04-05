# basicauth (Basic 認証バリデータ)

概要: `basicauth` パッケージは、Echo の Basic 認証ミドルウェア用に `echomw.BasicAuthValidator` を生成するユーティリティを提供します。主にメトリクス等の簡易保護に利用される想定です。

## 役割

- `NewBasicAuthValidator` により、`username`/`password` を検証する `echomw.BasicAuthValidator` を生成します。
- 検証ロジックは `config.MetricsConfig` に設定されたユーザ名とパスワードを照合するシンプルな実装です。

## 必要度

### 本番運用での必須度

- 必須度: 本番環境では任意（運用要件に依存）

理由: メトリクス等の軽量な保護手段として利用できますが、より堅牢なアクセス制御（ネットワーク ACL、プロキシ認証、OAuth/OIDC 等）を併用または代替することを検討してください。

### 開発/テスト運用での必須度

- 必須度: 開発/テストで推奨

理由: ローカルや CI でメトリクスのアクセスを制限したい場合に簡単に導入できます。設定は `MetricsConfig` で管理されます。

### 無効化した場合の影響

- 無効化すると、Basic 認証を期待するエンドポイント（例: メトリクス）への未認証アクセスが可能になります。代替の認証手段を用意していない場合、情報漏洩リスクが高まります。

## 注意点

- 検証は `MetricsConfig.UserName()` と `MetricsConfig.Password()` に依存します。これらの値は安全に管理してください（平文設定を避ける、Secret 管理を利用する等）。
- 認証失敗時は `apperror.ErrUnauthenticated` を返すため、呼び出し側のミドルウェアで適切にハンドリングしてください。
- より厳密な要件（IP 制限、レート制限、多要素認証など）がある場合は、本実装を拡張するか、別の認証方式を採用してください。

## 使い方（簡易）

ミドルウェア登録例（Echo）：

```go
e.Use(echomw.BasicAuth(basicauth.NewBasicAuthValidator(mtcCfg)))
```

---
