---
status: proposed        # proposed | accepted | superseded | deprecated
date: YYYY-MM-DD
deciders: []            # 決定者, 例: [maintainers]
supersedes:             # この ADR が置き換える ADR 番号（あれば, 例: 0003）
superseded-by:          # この ADR を置き換える ADR 番号（あれば）
# boilerplate-only:replace-begin
tags: []                # 例: [architecture, http]; exclusion は追加: setup-review
# boilerplate-only:replace-with
# = tags: []                # 例: [architecture, http]
# boilerplate-only:replace-end
---

# ADR-NNNN: 命令形の決定タイトル

## ステータス

proposed | accepted | superseded by [ADR-XXXX](XXXX-....md)

## 背景

どのような力が働いているか — 決定を必要とする問題・制約・目標。解決策ではなく *なぜ* を述べる。（**exclusion** ADR の場合は、意図的に除外する機能と、それを含める方向への圧力を述べる。）

## 決定

1〜2 文で選択内容を述べる。exclusion の場合:「我々は意図的に X を提供しない。」スコープと境界を具体的に。

## 影響

### ポジティブな影響

- ...

### ネガティブな影響

- ...

<!-- 任意: ### 中立な影響 -->

## 検討した代替案

### 代替案 A

検討した理由と却下した理由。

### 代替案 B

...

## 補足

設計ドキュメント・この決定を強制するルール（`../../../rules.md#...`）・関連 ADR へのリンク。任意。
