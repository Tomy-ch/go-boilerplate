# Domain Spec Format

`docs/spec/<feature>/domain.md` の節構成と自動派生ルール。

## 節構成

| 節 | 内容 |
| --- | --- |
| Overview | purpose を 1〜2 段落 |
| Entity | フィールド一覧（YAML: name / type / required / min/max など） |
| Cross-field Invariants | 複数フィールドの整合条件（箇条書き） |
| Behavior Methods | 状態遷移メソッド（YAML: signature / description） |
| Value Objects | VO 定義（YAML: underlying_type / validation / factory / methods、無ければ省略） |
| Repository Methods | IF メソッド（YAML: signature / behavior） |

## 自動派生ルール（spec に書かない）

| 項目 | 派生元 |
| --- | --- |
| Error 名（`ErrInvalid<Field>` 等） | Entity フィールド + cross-field invariants |
| Constant 名（`min<Field>Length` / `max<Field>Length` 等） | Entity フィールドの min/max |
| Field 識別子定数（`Field<Name>`）+ collect-all 検証 + `apperror.WithDetails` | ユーザーが修正できる入力フィールド（サーバ内部の不変条件は first-error のまま。ADR-0043） |
| Getter | Entity フィールド（pointer 型は `ptr.Copy`） |
| 単純な型検証 | Entity フィールドの type / required |
| ID `IsNil` 検証 | Entity の `id` フィールド |

## テンプレ例

```markdown
# User Management — Domain Spec

## Overview

TODO: aggregate の purpose を 1〜2 段落で記述。

## Entity

\`\`\`yaml
package: internal/domain/user
struct: User
fields:
  - name: TODO  # field name (camelCase)
    type: TODO  # Go type
    required: true
    # min_length / max_length / min / max を必要に応じて追加
\`\`\`

## Cross-field Invariants

- TODO: 例 `updatedAt >= createdAt`

## Behavior Methods

\`\`\`yaml
- name: TODO
  signature: TODO(arg Type) error
  description: TODO
\`\`\`

## Value Objects

\`\`\`yaml
# 利用しない場合は節ごと削除
- name: TODO
  underlying_type: TODO
  validation: TODO
  factory: NewTODO
  methods:
    - name: Value
      returns: TODO
\`\`\`

## Repository Methods

\`\`\`yaml
- name: TODO
  signature: TODO(ctx context.Context, ...) (TODO, error)
  behavior: TODO
\`\`\`
```
