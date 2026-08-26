---
status: accepted
date: 2026-08-18
deciders: [maintainers]
tags: [process, ai]
---

# ADR-0009: 永続的なエージェント状態を所有する正本の形に保つ

English canonical: [0009-long-running-agent-state.md](0009-long-running-agent-state.md)

## 決定

進捗ログをリポジトリ状態として蓄積しない。describing 文書を変える所見は、それを所有する README または `docs/` reference へ還元する。governing 文書との衝突は architect または tech lead へ提起する。per-run 状態は再開意味論を所有する skill 固有の gitignore された `tmp/` artifact に置く。`.agents/` へ移さず、`impl-issue`、`full-verify`、`full-apply` の異なる形式も統一しない。

[ADR-0008](0008-agent-environment-alignment.md) の閉じた改善ループは単一の run を越えて残る観測を必要とする。その観測は **リポジトリの外、issue tracker に置く** — `.agents/` ではない。保持するのは**圧縮された所見**、すなわち何にぶつかったか、それがどの制御に帰属するか、その制御への変更が測定上どう効いたかである。決して保持しないのは、エージェントが何をしたかの叙述である。判定基準は新しさではなく再利用可能性にある。圧縮された所見は後続の run と、制御を Keep / Simplify / Revise / Delete / Revert のどれにするか判断する人間に消費されるが、run ログは読み返せるだけである。

リポジトリが持つのはループの**設定**だけとする。スキーマ、閾値、除外リストの類であり、これらは意図して編集され、他のコミット対象と同じようにレビューされる。ローカルで生じた観測は skill が所有する gitignore された `tmp/` artifact に一時保持し、セッション終了時に issue へ送出する。通常の開発が作業ツリーを汚すことはない。

## 帰結

リポジトリにはエージェント活動の線形履歴ではなく、耐久する知識だけが残る。resume artifact は所有 workflow が必要とする間だけ存在する。ループの観測はそもそもリポジトリ状態にならない。セッション中も作業ツリーは汚れず、誤って記録した所見の訂正や撤回はコミットではなく issue の編集で済む。環境 hook は worktree、vendor tree、DB-slot marker を観測してよいが、slot を取得したり DB を再初期化したりしてはならない。後者は DB 作業開始時に意図して行う、破壊的になり得る操作である。

## 検討した代替案

### すべての run の進捗を永続化する

誰が読むか、どの権威を持つかが不明な第二の文書体系を無限に増やすため不採用。

### すべての resume 状態を `.agents/` に置く

`.agents/` はコミット対象の共有状態であり、resume artifact は per-run の operator 状態なので不採用。

### 既存の 3 形式を標準化する

承認済み計画との突合、冪等な review 列挙、apply / defer 判断は source of truth が異なるため不採用。

### ループの観測を `.agents/` に置く

却下。セッションのたびに作業ツリーが汚れ、所見の訂正や削除 — 制御が変われば所見は直ちに古くなるので、ループはこれを日常的に行う — にコミットが要る。維持に git 操作を要する保管場所は、維持されなくなる保管場所である。issue tracker は編集・削除・ラベル付け、および所見に対処した pull request との相互リンクを既に備えている。
