# 冪等性（Idempotency-Key）

[English](README.md) | 日本語

非冪等な書き込み（POST/PATCH/PUT）をクライアントの再送に対して安全にします。副作用は**高々1回**だけ実行され、再送には**同一の結果**を返します。

## 1. 概念 — なぜ tx だけでは不十分か

DB トランザクションは「**1リクエストの原子性**」しか守りません。**再送をまたいだ重複排除**（ネットワークタイムアウト・ダブルサブミット・クライアントの自動リトライ）は別問題です。自然な一意キーを持たない書き込み（`uuid.New()` 採番の POST・残高加算・課金・メール送信など）では、再送で副作用がもう一度走ってしまいます。

このパッケージは、クライアント供給の `Idempotency-Key` でその隙間を閉じます。

混同しないこと：

- **楽観ロック** … *別々の2操作*が同じ行を奪い合う lost update を防ぐ（`version` 列）。冪等性とは直交、併用可。
- **レートリミット** … エッジ（GW/LB）の責務で、本リポでは scope 外。

冪等性キーは「非冪等で再送されうる書き込み」**だけ**に付けます。`GET` に付けるのは無意味、課金エンドポイントに付け忘れるのはバグです。

## 2. 状態遷移

```text
(なし)
  │  INSERT ON CONFLICT DO NOTHING（業務 tx 内）
  ▼
claimed ──── 業務 fn 成功 + 結果保存（同一 tx）───▶ completed
  │                                                  │
  │ 業務エラー / クラッシュ → tx ロールバック          │ 再送
  ▼  （キー自動解放 → 再実行）                         ▼
(なし)                                        保存済み結果を replay

再送の分岐:
  - completed への再送   → replay（業務 fn は実行しない）
  - claimed への再送     → 409（処理中・後で再試行）
  - 指紋不一致           → 422（同キー・別リクエスト）
  - TTL 失効（GC 済）    → 新規実行
```

claim と業務書き込みは**同一 tx**（厳密整合）です。業務失敗は claim ごとロールバックされ、**失敗はキーを自動解放**します。

## 3. エンドポイントを冪等にする手順

opt-in の2ステップ（未採用のエンドポイントは挙動不変）：

1. **入り口 middleware** を handler の `NewStrictHandler` 第二引数へ差す：

   ```go
   gen.RegisterHandlers(e, gen.NewStrictHandler(server,
       []gen.StrictMiddlewareFunc{idempotencymw.StrictMiddleware[gen.StrictHandlerFunc]()}))
   ```

2. **ユースケース呼び出しを `Run[T]` で包む**（成功ステータスを渡す）：

   ```go
   dto, _, err := idempotency.Run(ctx, s.idem, http.StatusCreated, func(ctx context.Context) (user.UserView, error) {
       return s.uc.CreateUser(ctx, params)
   })
   ```

`PostUsers`（`internal/controller/handler/v1/users/v1_users_handler.go`）が参照採用です。middleware は `Idempotency-Key` ヘッダがある時だけ反応し、無ければ `Run` は `businessFn` を素通し実行します（非破壊）。

## 4. クライアント向け契約

- **`Idempotency-Key` ヘッダ**：非空 / ≤255 / 印字可能 ASCII（`0x21`–`0x7E`）。UUID 推奨・非必須。形式不正 → **400**。
- **replay**：同キー・同リクエストの再送は最初の結果を返す。
- **409**：そのキーは処理中（並行再送）→ 後で再試行。
- **422**：同キーを*別*リクエストボディで使い回した。
- 認証プリンシパル単位でスコープ（`UNIQUE (scope, idempotency_key)`）。他人のキーに衝突・覗きはできない。

## 5. (c) per-endpoint スコープ拡張（config フラグにはしない）

既定の scope = principal。エンドポイント単位でも隔離する（`scope = principal + operationId`）には、scope 組成の**1点**を変えるだけ。入り口 middleware が `operationId` を無料で持つ（o11y ラベルにも使用）。ランタイム config は作らず、コード改修として残す。onion 注：operationId は HTTP 由来なので、より純にするなら UC メソッドの identity を渡す案も可。

## 6. 運用

- **GC ジョブ** `idempotency-gc`（`internal/controller/job/idempotencygc/`）が失効行をバッチ削除。外部スケジューラから `cmd job idempotency-gc`（`--batch-size=N`、既定 10,000）で起動。推奨間隔は**毎時**（TTL 24h ゆえリアルタイム不要）。
- **TTL = 24h** = リトライ許容窓。TTL 経過後の再送は新規実行になる。
- **メトリクス**（`Deps.Metrics`: replay / conflict / fingerprint-mismatch カウンタ）は任意の拡張点。**既定は no-op**（観測性バックエンド未配線）で、配線時に `Deps.Metrics` へ実装を注入する。
- **オペレーションごとに成功ステータスは1つ**: `Run[T]` は `successStatus` を1つ記録し、`PostUsers` は常に 201 を返す。成功ステータスが複数あり得る（例: 200 と 201）エンドポイントに `Run[T]` を採用する場合は、保存ステータスで分岐するようハンドラを拡張すること（現状の replay はハンドラ固定のレスポンス型で再描画する）。

## セキュリティ / 保存の注意

- **scope 越境（IDOR）**：`scope` に DB の FK/RLS は無く、隔離は実装で担保。全クエリが `WHERE scope = <principal>` を持ち、`id` 単独 lookup を作らない。`Store` IF は scope を必須引数にする。
- **`response_payload` の PII**：結果 DTO を JSON(BYTEA) 保存する。PII を含む DTO（例 `UserResponse`）では DB ダンプ/バックアップの漏洩面に注意（24h TTL で消える）。気になる場合はキャッシュ専用 DTO / pgcrypto 暗号化 / 素の PII DTO を保存しない、のいずれかを選ぶ。
- **`request_fingerprint`** は構造上常に 32byte の SHA-256（middleware が `sha256(method+path+typed-request)` を生成）。DB の `CHECK (octet_length = 32)` を defense-in-depth として追加可。

## 配置

| 層 | パス |
| --- | --- |
| migration | `database/migrations/000001_create_idempotency_keys.*.sql` |
| sqlc DML | `database/dml/system_query/idempotency/` |
| boundary | `internal/usecase/boundary/idempotency/`（`Store`） |
| infrastructure | `internal/infrastructure/rdb/system_query/idempotency/` |
| usecase | `internal/usecase/idempotency/`（`Run[T]`, `GCUsecase`） |
| controller（入り口） | `internal/controller/httpstack/idempotency/` |
| GC ジョブ | `internal/controller/job/idempotencygc/` |
