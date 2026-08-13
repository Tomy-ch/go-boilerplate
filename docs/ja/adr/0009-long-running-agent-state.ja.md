---
status: accepted
date: 2026-08-10
deciders: [maintainers]
tags: [process, ai]
---

# ADR-0009: 永続的なエージェント状態を所有する正本の形に保つ

English canonical: [0009-long-running-agent-state.md](../../adr/0009-long-running-agent-state.md)

## 決定

進捗ログをリポジトリ状態として蓄積しない。describing 文書を変える所見は、それを所有する README または `docs/` reference へ還元する。governing 文書との衝突は architect または tech lead へ提起する。per-run 状態は再開意味論を所有する skill 固有の gitignore された `tmp/` artifact に置く。`.agents/` へ移さず、`impl-issue`、`full-verify`、`full-apply` の異なる形式も統一しない。

## 帰結

リポジトリにはエージェント活動の線形履歴ではなく、耐久する知識だけが残る。resume artifact は所有 workflow が必要とする間だけ存在する。環境 hook は worktree、vendor tree、DB-slot marker を観測してよいが、slot を取得したり DB を再初期化したりしてはならない。後者は DB 作業開始時に意図して行う、破壊的になり得る操作である。

## 検討した代替案

### すべての run の進捗を永続化する

誰が読むか、どの権威を持つかが不明な第二の文書体系を無限に増やすため不採用。

### すべての resume 状態を `.agents/` に置く

`.agents/` はコミット対象の共有状態であり、resume artifact は per-run の operator 状態なので不採用。

### 既存の 3 形式を標準化する

承認済み計画との突合、冪等な review 列挙、apply / defer 判断は source of truth が異なるため不採用。
