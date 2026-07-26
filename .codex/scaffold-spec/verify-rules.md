# Verify-Spec Rules

`verify-spec` 系 skill が実施する検証の粒度と cross-layer / 派生元参照チェック方針。lean A 構成（domain.md + usecase.md の 2 spec）前提。

## 検証スコープ

lean A では controller / infra に spec を持たないため、`verify-spec` の責務は:

1. **format 検査** — 必須節の有無、YAML パース可否
2. **派生元との整合性** — spec が SQL / OpenAPI / sqlc gen など外部入力と矛盾しないか
3. **cross-spec 参照整合性** — `usecase.md` の `calls:` が `domain.md` Repository / Behavior / boundary に存在するか

## 層ごとの責務

| skill | 担当 spec | 検査内容 |
| --- | --- | --- |
| `spec-validator-domain` | `domain.md` | format（節 / YAML パース）+ Entity フィールド ↔ `database/migrations/` の `CREATE TABLE` カラム突き合わせ + 内部整合性（Behavior / VO / Repository methods） |
| `spec-validator-usecase` | `usecase.md` | format + cross-spec 参照（domain.md Repository / Behavior / boundary） + 命名規約（動詞接頭辞、Usecase interface 命名 — `internal/usecase/README.md` + sibling pkg 由来）+ Workflow 内部整合性（tx_required + boundary calls） |
| `verify-spec` | 統合 | 存在する spec を検出して上記を chain |

**依存方向に関する注意**: `spec-validator-usecase` は OpenAPI operationId カバレッジを **検査しない**。これは依存方向逆転（usecase は HTTP/OpenAPI を知らない、知るべきでない）になるため。OpenAPI ↔ usecase mapping の検証は **`scaffold-controller` の責務**（scaffold 時に「operationId に対応する usecase method があるか」を確認、無ければ scaffold 失敗で hand-off）。実装後の handler ↔ operationId 一致は `arch-check`（controller 監査）が担当。

## cross-spec 参照ルール（lean A）

| 参照 | チェック方法 |
| --- | --- |
| `usecase.md` `calls: <aggregate>.Repository.<Method>` | `domain.md` の Repository Methods 一覧に存在 |
| `usecase.md` `calls: <aggregate>.<BehaviorMethod>` / `<aggregate>.New` | `domain.md` の Behavior Methods / Entity factory に存在 |
| `usecase.md` `calls: <boundary>.<Method>` | `usecase.md` の Dependencies に boundary 名が存在（Method 自体は Go compiler 任せ） |

## 派生元との整合性ルール（lean A 新規）

| 派生元 | 検査 | 担当 skill |
| --- | --- | --- |
| `database/migrations/<latest>.sql` の `CREATE TABLE` | `domain.md` Entity フィールド名（snake_case → camelCase）+ Go 型マッピングが対応 | spec-validator-domain |
| `internal/usecase/README.md` + sibling pkg | `usecase.md` Interface 命名規約（動詞接頭辞、Usecase interface 命名）に準拠 | spec-validator-usecase |
| OpenAPI gen (`internal/controller/handler/<path>/gen/server.gen.go`) | `ServerInterface` operationId ↔ usecase method の mapping 確立 | **scaffold-controller**（verify-spec ではなく、scaffold 時に検査。依存方向に整合） |
| sqlc gen (`internal/infrastructure/rdb/sqlc/gen/*.gen.go`) | Repository method ↔ sqlc 関数の対応 | **scaffold-infra-db**（compile 時 catch も併用） |

これらは scaffold-X skill が input として直接読むため、spec / 既存 doc が drift していると scaffold が崩れる。spec 内部の整合は verify-spec が、scaffold 入力との整合は scaffold-X 自身が scaffold 時に検査する役割分担。

## 検出粒度（参考）

**b2 採用**: format + cross-layer 参照整合性。**b3（sqlc gen 関数の存在検証）** は対象外 — compile 時に catch されるため。

## scaffold-X skill 内では pre-check しない

scaffold-X 内では入力の有効性を仮定する。事前検証は `verify-spec` の責務、事後検証は `make test` の責務。`scaffold-endpoint` から起動された場合は `verify-spec` が違反検出時に下流 chain を中断する。

## 参照する spec format

各 layer の必須節 / YAML schema は以下を参照（実行時に読み込む）:

- `domain-spec.md`
- `usecase-spec.md`
