# Usecase Spec Format

`docs/spec/<feature>/usecase.md` の節構成と自動派生ルール。

## 節構成

| 節 | 内容 |
| --- | --- |
| Overview | purpose |
| Interface | usecase 名 + メソッドシグネチャ |
| DTOs | 入出力 struct のフィールド |
| Dependencies | clock / tx.Manager / encrypter / Repository 等の依存リスト |
| Workflow | メソッドごとの呼び出し手順（YAML: tx_required / steps / calls / errors） |

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
```
