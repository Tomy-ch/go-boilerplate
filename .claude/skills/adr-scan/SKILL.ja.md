> このファイルは `SKILL.md`（canonical / 英語）の日本語参考訳です。スキルとしては読み込まれません（参考用）。

# adr-scan (provisional)

**Status: temporary.** 一度きりの `decisions.md` → `docs/adr/` 移行調査を駆動するために作られた。ADR の
スケルトン / 採番が合意されたら削除すること。read-only — `docs/adr/**` を書いたりソースを編集したりせず、
候補 inventory を出力するだけ。

## Purpose

`docs/decisions.md` には 8 件の明示的な decision があるが、architectural decision は design docs / rules /
意図的な exclusion / コードコメントにも散在している。正式な per-file ADR セットへ移行する前に、ADR 相当の
全 surface を洗い出し、採番と分割を一度で済ませる。

## Classification taxonomy (the core judgment)

各候補は必ず次のいずれか 1 つ:

- **decision** — 恒久的な帰結を伴う、複数案の中からの選択（X over Y）。ADR-worthy。
- **exclusion** — 「意図的に X を行わない」という明確な判断と根拠。ADR-worthy（negative decision）。
- **rule** — 日々運用で強制される帰結 / 制約（layer deps, DTO boundary）。`docs/rules.md` に留まる。ADR を *参照* してよいが、それ自体は ADR ではない。
- **inventory** — コードとともに drift するカタログ（dependency table）。living reference doc に留まり、ADR にはしない。

候補が ADR-worthy になるのは次を満たすときのみ: 検討された alternatives を持つ（あるいは含意する）、cross-cutting
または reverse しにくい、かつ単に rule/inventory を言い換えただけではない。

## Scan surfaces (fan out one worker each)

1. **Canonical** — `docs/decisions.md`, `docs/architecture.md`, `docs/rules.md`。明示的な decision + rule に埋め込まれた decision を抽出し、decision と rule を分離する。
2. **Subsystem design** — `docs/design/*.md`（idempotency / job / observability / outbox / rest / worker）。各 design doc の「why / alternatives / trade-off」の内容。
3. **Deliberate exclusions** — `docs/project/*.md`（特に `out-of-scope.md`）, `policy.md`, `scope.md`。根拠を伴う negative decision。
4. **Latent** — per-package `README.md`（Design Intent / Notes）+ コードの `// why` rationale コメント。コード内で下されたが doc に昇格していない decision。

## Output (per candidate)

```text
- title:        short decision title (imperative, ADR-style)
  type:         decision | exclusion | rule | inventory
  adr_worthy:   yes | no
  source:       file:line (evidence)
  alternatives: present | implied | none
  proposed_adr: which ADR bucket it belongs to (or existing 0001..N)
  note:         one line — why worthy / why not
```

次にまとめる: (A) 提案採番付きの ADR bucket list, (B) rules-stay list,
(C) inventory-stay list, (D) decisions.md に無い新規発見の decision。
