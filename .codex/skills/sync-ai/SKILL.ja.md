> このファイルは `SKILL.md`（canonical / 英語）の日本語参考訳です。スキルとしては読み込まれません（参考用）。

# AI スキル同期（Codex）

スキルを生のディレクトリコピーではなく、**片方向の意味的移植**として同期する。今回だけ送信元スキルを正典とし、受信側スキルは各 AI 環境にネイティブな形を保つ。

`manage-skill` は、新規 Codex スキルの作成または大きな変更後、そのスキルが Codex 専用でない限りこの手順を起動する。Claude 起点の変更も、Claude を送信元として同じ流れで扱う。

## 1. 方向と単位を確定する

スキル名を 1 つ、方向を 1 つだけ確定する。

```text
source: .claude/skills/<name>/  → target: .codex/skills/<name>/
source: .codex/skills/<name>/   → target: .claude/skills/<name>/
```

ユーザーが送信元を明示した場合だけ推定する。ファイル時刻から方向を推定してはならない。両方が変更されている場合は、どちらを正典とするか確認して停止する。2 つの編集済みスキルを自動マージしない。

送信元一式と、存在する場合は受信側一式を確認する。

```sh
git diff -- .claude/skills/<name> .codex/skills/<name>
find .claude/skills/<name> -type f | sort
find .codex/skills/<name> -type f | sort
```

`SKILL.md`、`SKILL.ja.md`、UI メタデータ、scripts、references、assets を 1 単位として扱う。転送メモは `tmp/sync-ai/` に一時的に残すだけとし、永続的な第 3 コピーや同期 manifest は作らない。

## 2. 転送契約を作る

送信元スキルから以下を抽出する。

- 目的、明示的な起動条件・非起動条件
- 必要な入力、承認、副作用、検証
- 再利用する scripts、references、assets
- コピーではなく変換が必要なプラットフォーム固有機構

送信元の項目を **port**、**adapt**、**omit** に分類する。例として、Claude の `AskUserQuestion` / `Agent` の指示は Codex のユーザー入力 / delegation 機構へ変換する。Claude 専用の設定、hook、権限構文は Codex に対応物がない限り省く。送信元固有のツール名を、対応確認なしに受信側へ持ち込まない。

## 3. 受信側ネイティブのワークフローで反映する

対象が `.codex/skills/` の場合、この転送契約で `manage-skill` を起動する。そこで以下を行う。

1. 受信側スキルをその場で作成または更新する。
2. 受信側固有の frontmatter と `agents/openai.yaml` を維持する。
3. 公式 skill-creator の検証フローを使う。
4. 英語の canonical `SKILL.md` から `SKILL.ja.md` を同期する。

対象が `.claude/skills/` の場合は、同じ転送契約を Claude の `/manage-skill` に渡す。Codex から Claude 固有のコマンド構文を書き込もうとしてはならない。

移植成功後も送信元スキルを削除しない。commit、push、公開、無関係なスキルの変更をしない。

## 4. 検証と報告

受信側環境の構造検証を実行し、受信側の差分を確認する。送信元の振る舞いを保ちつつ、受信側がサポートするツールと設定だけを使っていることを確認する。

方向、送信元の commit／差分基準、port/adapt/omit の判断、変更ファイル、受信側で表現できない意図を報告する。

## ガードレール

- 送信側の `manage-skill` が同期を開始する。受信側の `manage-skill` は子操作として扱い、再び外向きの同期を開始しない。
- 特定プラットフォーム専用スキルは専用のままにする。弱い移植を強行せず理由を記録する。
- 双方向自動同期、タイムスタンプによる競合解決、ディレクトリ全上書きを禁止する。
- 転送成果物は gitignore された `tmp/` に置く。保守対象は 2 つのネイティブなスキルディレクトリだけとする。
