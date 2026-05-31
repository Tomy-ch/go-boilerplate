> このファイルは `SKILL.md`（canonical / 英語）の日本語参考訳です。スキルとしては読み込まれません（参考用）。

# Verify Spec

spec 検証の統合スキル。`docs/spec/<feature>/` 配下に存在する spec ファイルに応じて per-spec skill を chain。

## 使うとき

- `scaffold-endpoint` 起動前の spec 不整合検知（`scaffold-endpoint` が自動 chain）
- spec 編集後の全 check 確認
- spec 作成中のクイックチェック

単一 spec のみ気にする時は per-spec skill（`verify-spec-domain` / `verify-spec-usecase`）を直接呼ぶ。

以下の用途には使いません:

- 生成コードの検証 — `make test`
- 実装 ↔ spec drift — `arch-check`
- 不整合の修正 — read-only、レポートのみ

## chain される per-spec skill

| skill | spec | チェック内容 |
| --- | --- | --- |
| `verify-spec-domain` | `docs/spec/<feature>/domain.md` | format + entity ↔ SQL soft + 内部整合性 |
| `verify-spec-usecase` | `docs/spec/<feature>/usecase.md` | format + cross-spec to domain + 命名規約 + Workflow 整合性 |

lean A 構成では controller.md / infra.md は存在しないため、それらの spec 検証は不要（controller / infra は実装時に OpenAPI + sqlc gen から導出され、verify は `arch-check-controller` / `arch-check-infra` が implementation 側で実施）。

## 最初のステップ: 対象 feature 確認

`AskUserQuestion` を起動直後に必ず呼ぶ（`scaffold-endpoint` から呼ばれて context に feature 名がある場合は除く）:

- 質問: 「検証対象の feature 名を選んでください」
- 選択肢: `docs/spec/` 直下のサブディレクトリを列挙 + 規約外パス用のフリーテキスト

feature ディレクトリが無い or spec ファイル無い時は明確メッセージで中断。

## Step 1. 存在する spec ファイル検出

確認済み feature について、以下の存在を確認:

- `docs/spec/<feature>/domain.md`
- `docs/spec/<feature>/usecase.md`

両方なし → メッセージ出して中断。

片方のみ → 該当 per-spec skill のみ chain。cross-spec チェック（例: usecase → domain refs）は `usecase.md` 単独存在時に「domain.md not found」として surface される。

## Step 2. per-spec skill chain

順序（domain 先行、usecase が参照するため）:

1. `domain.md` 存在 → `verify-spec-domain` を feature 名供与で起動
2. `usecase.md` 存在 → `verify-spec-usecase` を feature 名供与で起動

各 child は findings（violation + suggestion 数）を返す。spec ラベル付きで収集。

## Step 3. 集約レポート（日本語）

```text
verify-spec 統合結果（feature: <feature>）

[domain] violations: N, suggestions: K
  - <verify-spec-domain の findings>

[usecase] violations: N, suggestions: K
  - <verify-spec-usecase の findings>

総計: violations <sum>, suggestions <sum>
```

全 clean:

```text
verify-spec 統合結果（feature: <feature>）
全 spec で違反は検出されませんでした（チェック済み: <spec list>）。
```

## Step 4. クロージング

- **単独実行**: レポート出力して exit。違反があっても exit 0（情報的）
- **`scaffold-endpoint` から chain**: `violations > 0` のとき親 skill が下流 chain を中断。「scaffold can not safely proceed」を明示

## AI 修正スコープ

完全 read-only。spec / source ファイル一切触らない。全 check は per-spec skill に委譲（child も read-only）。

## 制約事項

- ❌ ルールをハードコード — 必ず per-spec skill に委譲（child が `.claude/scaffold-spec/<layer>-spec.md` + `verify-rules.md` から読む）
- ❌ 違反の自動修正
- ❌ ファイル変更
- ❌ 対象確認 `AskUserQuestion` をスキップ
- ❌ 対象 spec ファイルが無い per-spec skill を chain
- ✅ 日本語集約レポート
- ✅ 存在する spec のみ chain
- ✅ per-spec skill が独立 standalone 動作可能であることを維持
- ✅ 全 per-spec チェックを 1 パスで実施（fail-fast しない）

## チェックリスト

- [ ] 対象 feature を `AskUserQuestion` で確認（または `scaffold-endpoint` から供与）
- [ ] 存在する spec ファイル検出（domain.md / usecase.md）
- [ ] 存在 spec のみ per-spec skill を chain
- [ ] 各 child が自身の検証を実施
- [ ] 集約日本語レポート出力
- [ ] ファイル変更なし
