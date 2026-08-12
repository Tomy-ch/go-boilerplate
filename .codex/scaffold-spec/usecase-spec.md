# Usecase Spec Format

`docs/spec/<feature>/usecase.md` の節構成と自動派生ルール。

## 節構成

| 節 | 内容 |
| --- | --- |
| Overview | purpose |
| Interface | usecase 名 + **全メソッド**のシグネチャ |
| DTOs | 入出力 struct のフィールド |
| Dependencies | clock / tx.Manager / encrypter / Repository 等の依存リスト |
| Workflow | メソッドごとの呼び出し手順（YAML: tx_required / steps / calls / errors） |
| Notes | 上の節に収まらない補足（定数の出自・索引の判断など）。無ければ節ごと省略 |

節構成表にない `##` 節を足してよい。ただし足せるのは **usecase の語彙で書かれた補足**に限る。
HTTP のワイヤ契約（クエリパラメータ・ステータスコードの一覧）は usecase が知ってよいものではないので
ここには置かない — 正本は `openapi/openapi.yaml` であり、依存方向の理由は
[`verify-rules.md`](verify-rules.md) が述べている。

## メソッドの書き方は 2 形式ある

同じ内容を書く場所が 2 つあり、メソッド単位でどちらかを選ぶ。

### 集約形（既定）

`## Workflow` の下にメソッドを並べる。ほとんどの feature はこれだけで済む。メソッドの識別子は
`### <Method>` の見出し、または 1 つの YAML リストにまとめる場合は各要素の `method:` キー。

### 展開形（`## <メソッドの説明>` を独立させる）

そのメソッドに、`steps:` の 1 行に収まらない散文（配置判断の根拠・他の経路との切り分け・ADR 参照）が
要る場合は、メソッドごとに独立した `##` 節を立て、散文と YAML を並べる。YAML は 1 ブロックにまとめ、
`input` / `output` / `dependencies` / `workflow`（`tx_required` / `steps` / `errors`）を持つ。
cursor ページネーションを持つ経路は `cursor`（`boundary` / `keys`）を足す。

どちらの形式でも次の 2 つは崩さない。

- **`## Interface` には全メソッドを列挙する。** 展開形で書いたメソッドを Interface から省くと、
  spec を機械的に読む側からはそのメソッドが存在しないのと同じになる。
- **`calls:` は集約形の検査点である。** 展開形は `calls:` を持たず、代わりに節スコープの
  `dependencies:` を持つため、cross-spec 参照の解決は依存名の粒度までしか効かない
  （[`verify-rules.md`](verify-rules.md)）。散文が要らないメソッドを展開形で書くと、
  得るものがないまま検査だけが粗くなる。

## 自動派生ルール（spec に書かない）

| 項目 | 派生元 |
| --- | --- |
| `usecase` struct | Interface + Dependencies |
| Constructor | Dependencies |
| Tracer 配線 | Workflow（メソッドごとに span） |
| DI 登録 | Interface 名 |
| 標準 error 伝播 | Workflow の errors |

## テンプレ例

```markdown
# User Management — Usecase Spec

## Overview

TODO: usecase の purpose を 1〜2 段落。

## Interface

\`\`\`yaml
package: internal/usecase/user
name: Usecase
methods:
  - name: TODO
    signature: TODO(ctx context.Context, input TODOInput) (TODOOutput, error)
\`\`\`

## DTOs

\`\`\`yaml
- name: TODOInput
  fields:
    - name: TODO
      type: TODO
- name: TODOOutput
  fields:
    - name: TODO
      type: TODO
\`\`\`

## Dependencies

\`\`\`yaml
# 利用しない depend は削除
- clock          # boundary.Clock
- tx_manager     # boundary.TxManager
- user_repository  # domain/user.UserRepository
\`\`\`

## Workflow

### TODO

\`\`\`yaml
tx_required: true
steps:
  - TODO
calls:
  - user_repository.TODO
  - clock.Now
errors:
  - TODO
\`\`\`

## TODO（展開形。散文が要るメソッドだけ節を立てる）

TODO: この経路をこう置いた理由・他の経路との切り分け・ADR 参照。

\`\`\`yaml
input:
  - TODO: TODO

output:
  struct: TODOView
  fields:
    - name: TODO
      type: TODO

cursor:                  # keyset ページネーションを持つ経路のみ
  boundary: todoCursor
  keys: [TODO, id(UUID)]

dependencies:
  - user_repository      # domain/user.UserRepository

workflow:
  tx_required: false
  steps:
    - TODO
  errors:
    - TODO
\`\`\`

## Notes

- TODO: 上の節に収まらない補足。無ければ節ごと削除
```
