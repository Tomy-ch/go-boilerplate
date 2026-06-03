> このファイルは `SKILL.md`（canonical / 英語）の日本語参考訳です。スキルとしては読み込まれません（参考用）。

# Scaffold Endpoint

オニオンアーキ準拠 endpoint を全層 chain で構築するトップレベルオーケストレータ。OpenAPI + SQL + spec 入力が揃った後に新規 feature を end-to-end で立ち上げる際に使用。

## 使うとき

- 2 spec（domain.md + usecase.md）+ OpenAPI YAML + SQL 用意済みで新規 feature を end-to-end 構築
- 全層を同じ規約で一貫構築 + 1 つの統合 FB レポートを最後に得たい

以下の用途には使いません:

- 既存 feature の変更 — 該当 layer skill (`scaffold-domain` / `-infra-db` / `-usecase` / `-controller`) を単独実行
- 入力アーティファクト（OpenAPI YAML、SQL、spec）の bootstrap — 人間が事前に書く前提

## 読み書き範囲

**読み込み（常時）**:

- `docs/spec/<feature>/{domain,usecase}.md`（child skill 経由、lean A: 2 spec のみ）
- `.claude/scaffold-spec/lifecycle.md` — canonical workflow / scaffold execution order

**書き込み**: 直接はなし。全書き込みは child skill 内、各自のスコープ内で発生。

## 前提条件（起動前に満たすべき）

| # | 前提 | 検証者 |
| --- | --- | --- |
| 1 | `domain.md` + `usecase.md` が `docs/spec/<feature>/` 配下に存在（lean A: 2 spec のみ） | verify-spec |
| 2 | spec format 有効 + cross-layer 参照整合 | verify-spec (Step 2) |
| 3 | OpenAPI YAML 書き込み済み + `make gen-api` で `internal/controller/handler/<path>/gen/` 生成済み | scaffold-controller 前提 |
| 4 | `database/dml/...` 配下に SQL ファイル書き込み済み + `make gen-query` で新 repository 用の sqlc gen ファイル生成済み | scaffold-infra-db 前提 |
| 5 | DB 起動中 + migrate + seed 済み（DB スキーマが新 SQL と一致） | 手動: `make serve` で環境起動 → `make db-init` |
| 6 | usecase spec が依存する boundary interface が `internal/usecase/boundary/` 配下に存在 | scaffold-usecase 前提 |

前提未充足時は該当 child skill が surface、本 skill が chain 中断。

> **環境に関する注記（前提 3〜5）:** `make gen-query` は `pg_dump` で稼働中の DB スキーマをダンプするため、**DB が起動している必要がある**（未起動だと `make gen-query` / `make test` が `could not translate host name "database"` で失敗する）。環境は**生 `docker compose` ではなく専用 make ターゲット**で起動すること: `make serve`（development プロファイル、`database` サービス含む）→ **`make db-init`**（local/test 両 DB を migrate **かつ seed**。テストは seed 前提のため、`db-*-migrate-up` 単体では不十分）→ その後に `make gen-query` / `make gen-api`。
>
> **ツールチェーンに関する注記（最終 `make fix` / `make test`）:** `make fix` や `make lint` が**ツールのバージョン不整合**（例: `golangci-lint` の "you are using a configuration file for golangci-lint v2 with golangci-lint v1"）で失敗した場合は回避策を取らず、`make install-tools` でローカルのツールを `tools.yaml` 固定バージョンに揃えてから再実行する（`tools.yaml` 自体を変更した場合は先に `make sync-tools`）。`PATH` の手動書き換えやバージョン指定バイナリの直叩きで代替しないこと。

## 最初のステップ: feature 確認

`AskUserQuestion` を起動直後に必ず呼ぶ:

- 質問: 「対象 feature 名 (kebab-case)」
- フリーテキスト。`docs/spec/<feature>/` 存在 + `domain.md` + `usecase.md` 揃いを検証（lean A 最小要件）

feature ディレクトリ無し or spec 不完全時は `/new-spec` を欠落 layer 分実行するよう案内。

## Step 1. spec 検証（自動 chain）

`verify-spec` skill を feature 名指定で起動。`verify-spec` が `violations > 0` を報告したら chain 中断:

```text
scaffold can not safely proceed: verify-spec で <N> 件の違反が検出されました。
spec を修正してから再度 /scaffold-endpoint を実行してください。
```

warning のみなら継続（warning は block しない）。

## Step 2. 依存順序で child skill chain

各 child skill を順に起動、feature 名を context として渡し各 child が spec パスを自動解決できるように:

1. **`scaffold-domain`** — entity + Repository IF + VOs + constants + errors + tests（+ mock 用 `make gen-api`）
2. **`scaffold-infra-db`** — sqlc gen ラップの Repository 実装（事前 `make gen-query` 実行済みを内部検証）
3. **`scaffold-usecase`** — Application Service + DTOs + tests（+ Usecase mock 用 `make gen-api`）
4. **`scaffold-controller`** — `ServerInterface` 実装 handler + tests

child skill 間で成否ステータスを伝播:

- **child 成功** → 次へ
- **child 失敗** → chain 停止、child の FB summary を surface、進めない

各 child skill は独立に:

- 自身の plan 確認 `AskUserQuestion` を実施（layer ごとに user 判断を確保）
- 自身の test 観点 subagent を起動
- 必要なら `make gen-api` 実行
- 自身のファイルを書き込み
- 書き込み後 `make fix` + `make test` 実行
- 失敗時 TODO + FB を surface

将来 "完全 unattended モード" が必要なら `--auto-approve` フラグを追加可能 — ただし既定は layer ごと確認で human-in-the-loop を維持。

## Step 3. 最終統合検証

全 4 child skill 成功後、統合最終検査:

```sh
make fix
make test
```

cross-layer 統合（handler → usecase → domain → infra）が全体としてコンパイル / テスト通るか確認。本 scaffold が触った 4 パッケージのカバレッジ行を surface。

最終ステップで `make test` 失敗時（child が自身でテスト済みなのでまれ）は TODO + FB で surface して停止。

## Step 3.5. ランタイム動作確認（curl + o11y）

Step 3 の `make test` は **usecase / repository をモック**するため、実際の Fx グラフ・HTTP ミドルウェア（認証 / OpenAPI バリデーション）・DB を通らない。実機でしか出ないバグ群がある: `security:` 宣言漏れ（認証なしで到達）、`BindHandler` の未登録 / 配線ミス、DI provider 不整合、実 DB での SQL フィルタ挙動差など。**curl はここでやるのが正しい** — 全層 + DI がここで初めて揃う。per-layer スキルでは不可（下位層 / DI が無く起動すらしない）。

前提:

- `make serve` 稼働中で `api_server` ログが `[Fx] RUNNING` 到達（`scaffold-controller` の DI 起動確認）。
- `make db-init` 実行済み（local + test を seed する正準セットアップ）。

手順:

1. **既知状態の対象を用意する。** 既存行が要るなら seed 済み id を使うか、作成エンドポイントで先に作る。資格情報 / 状態依存の確認（例: パスワード変更）は、平文 / 状態を自分が把握する行を作る（ローカル検証なら既知 bcrypt ハッシュの `psql` insert でも可）。
2. **新エンドポイントを curl**（ローカル認証は `Authorization: Bearer debug:<subject>`）し、以下を確認:
   - ルート到達 — ルータ 404 でない応答 = ハンドラ登録 / DI 配線 OK の証拠；
   - 正常系が期待どおりの status / body；
   - 主要異常系: NotFound (404)、バリデーション (400/422)、**`security:` 宣言があるなら** no-token ⇒ 401（保護が効いているか）；
   - 変更が**実際に反映**されたか（新状態で再取得 / 再認証）。2xx だけで満足しない。
3. **o11y ログを 1 回確認**: 1 リクエスト分のトレースが全層（controller → usecase → infra）を貫き、発行 SQL が想定どおりかを見る。この 1 回の確認後は、再確認を curl ではなく o11y に頼れる。

破壊的ガード: curl がデータを変更し、元に戻す手段が `make db-init`（等）しかない場合は、**実行前にユーザー確認**（`CLAUDE.md` 準拠）。検証用に作成した行は後始末する。

いずれか失敗時は TODO + FB を surface して停止（コミットしない）。

## Step 4. クロージング

日本語サマリ:

```text
scaffold-endpoint 完了（feature: <feature>）。

  ✓ verify-spec: violations 0
  ✓ scaffold-domain: <N> ファイル作成、coverage 100%
  ✓ scaffold-infra-db: <N> ファイル作成、coverage <X>%
  ✓ scaffold-usecase: <N> ファイル作成、coverage 100%
  ✓ scaffold-controller: <N> ファイル作成、coverage 100%
  ✓ make test: 全体 OK
  ✓ ランタイム動作確認: curl 到達 / 認証 / 主要異常系 / o11y トレース OK

次のアクション:
  - /commit で 4 層 + DI 変更をコミット
  - /submit-pr で PR 作成
```

いずれかのステップ失敗時は失敗ステータス表 + 失敗 child の FB を出力、user が fix-forward を判断。

commit しない。push しない。

## AI 修正スコープ

本 skill 自体はファイルを書かない。全スコープは child skill に委譲（各 SKILL.md の constraint 参照）。

## 制約事項

- ❌ ソースファイルを直接変更（child skill に委譲）
- ❌ `verify-spec` (Step 1) をスキップ — spec 整合性の safety net
- ❌ 失敗した child skill を素通り — 停止して FB surface
- ❌ 後段 child 失敗時に成功済み earlier child の書き込みを自動 rollback — user 判断
- ❌ feature 確認 `AskUserQuestion` をスキップ
- ✅ ユーザー向け出力は日本語
- ✅ 依存順序（domain → infra-db → usecase → controller）で child 起動
- ✅ 5 chained step + 最終 `make test` を統合した最終レポートを surface
- ✅ 各 child skill が自身の確認 layer ごとに取る（judgment-heavy step で human-in-the-loop）
- ✅ ランタイム curl + o11y 確認（Step 3.5）を実施 — `make test` だけでは DI / ミドルウェア / DB を通らない
- ✅ 元に戻す手段が `make db-init` しかない破壊的 curl は実行前にユーザー確認

## チェックリスト

- [ ] feature 名を `AskUserQuestion` で確認
- [ ] `verify-spec` 実行、違反時は chain 中断
- [ ] `scaffold-domain` 成功実行（または失敗時 chain 停止）
- [ ] `scaffold-infra-db` 成功実行（または失敗時 chain 停止）
- [ ] `scaffold-usecase` 成功実行（または失敗時 chain 停止）
- [ ] `scaffold-controller` 成功実行（または失敗時 chain 停止）
- [ ] 全 child 成功後に最終 `make fix` + `make test` 実行
- [ ] ランタイム動作確認（Step 3.5）: curl 到達 / 認証 / 主要異常系 / o11y トレースを確認
- [ ] layer ごとのファイル数 + カバレッジを含む統合日本語サマリ
- [ ] commit / push なし
- [ ] child 失敗時に書き込み済みファイルの自動 rollback をしていない
