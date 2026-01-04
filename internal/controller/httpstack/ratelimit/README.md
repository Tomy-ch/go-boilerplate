# rate limit ミドルウェア

概要: このパッケージは IP アドレス単位でのレート制限（rate limiting）を提供します。Echo 用のミドルウェア `Middleware(rl, cfg)` と、IP ごとのレートリミッター実装 `NewIPRateLimiter(cfg)` を含みます。設定により有効/無効を切り替えられ、不要な IP エントリは TTL に応じてクリーンアップされます。

## 役割

- リクエスト送信元 IP ごとに `golang.org/x/time/rate` の `rate.Limiter` を割り当て、許容量を超えたアクセスに対して 429 応答を返す。
- Echo ハンドラ向けのミドルウェアを提供し、設定で無効化した場合は何も処理せず次のハンドラへフォワードする。
- IP エントリの自動クリーンアップ（TTL ベース）を提供することでメモリ使用量を抑える。

## 必要度

### 本番運用での必須度

- 必須度: 本番運用で推奨

- 理由: 外部からの過剰アクセスや DoS の緩和、API 利用制御のためにレート制限は本番環境で推奨されます。設定次第で無効化可能です。

### 開発/テスト運用での必須度

- 必須度: 開発/テスト運用で任意

- 理由: 開発環境ではしばしばレート制限を無効化してテストを行いますが、レート関連の動作検証が必要な場合は有用です。テスト用にモック `mock/mock_ip_rate_limiter.go` が用意されています。

### 無効化した場合の影響

- ミドルウェアを無効化すると、同一 IP からの短時間大量リクエストを制限できなくなり、API サーバーが過負荷に陥るリスクが高まります。

## 注意点

- ミドルウェアの有効/無効は `config.IPRateLimitConfig.Enabled()` によって制御されます。設定が無効の場合、`Middleware` はノーオペレーションとして動作します。
- クライアント IP は `echo.Context.RealIP()` で取得します。リバースプロキシなどを挟む環境では正しく設定されているか確認してください（X-Forwarded-For 等）。IP が取得できない場合は `unknown` をキーとして扱います。
- `NewIPRateLimiter` の内部はスレッドセーフだが、`Cleanup()` の呼び出しタイミングは外部で管理してください（定期ジョブや GC タイミングで呼ぶ想定）。

## 実装の要点

- `Middleware(rl, ipCfg)`: `ipCfg.Enabled()` が false の場合はそのまま次のハンドラへ。許可されないリクエストには `Retry-After: 1` ヘッダをセットし `apperror.ErrTooManyRequests` (HTTP 429 相当) を返す。
- `IPRateLimiter` インターフェース: `AllowRequest(c echo.Context) bool` と `Cleanup()` を提供。
- `NewIPRateLimiter(cfg)`: IP ごとに `rate.Limiter` を生成・キャッシュし、最後に観測した時刻を保持する `limiterEntry` を管理する実装を返す。`ensureLimiter` でレートリミッターを作成または更新し、`Cleanup` で TTL 超過のエントリを削除する。

## 使い方

- ミドルウェア登録例（Echo）:

```go
rl := ratelimit.NewIPRateLimiter(ipLimitCfg)
e.Use(ratelimit.Middleware(rl, ipLimitCfg))
// 定期的に rl.Cleanup() を実行して古いエントリを削除
```

## 前提 / 要件

- `config.IPRateLimitConfig` により `Limit()`, `Burst()`, `TTL()`, `Enabled()` が提供されること。
- Echo の `RealIP()` が環境に合わせて正しくクライアント IP を返すよう、プロキシ設定（`TrustedProxies` 等）を調整してください。

## トラブルシューティング

- 期待どおりに制限が効かない: `ipLimitCfg` の `Limit`/`Burst` 値や `Enabled` フラグを確認してください。
- 同一 IP なのに別扱いになる: リバースプロキシの設定で `RealIP()` が期待するヘッダを参照しているか確認してください。
- メモリ増加: `Cleanup()` を定期実行せずに長時間稼働すると `entries` が肥大化します。定期的に `Cleanup()` を呼ぶ仕組みを導入してください。
