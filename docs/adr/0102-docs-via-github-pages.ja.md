---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [docs, deploy]
---

# ADR-0102: docs/ の静的コンテンツを GitHub Pages で公開（production プッシュ時にリリース）

English canonical: [0102-docs-via-github-pages.md](0102-docs-via-github-pages.md)

## ステータス

accepted

## 背景

このリポジトリは `docs/` ディレクトリをアーキテクチャドキュメント・設計上の意思決定・API リファレンス・ガイドの正典として管理している（ADR-0010 参照）。このコンテンツは、生の Markdown としてリポジトリで閲覧するだけでなく、レンダリングされた形式でブラウズできると、コントリビューターや読み手にとって価値が高い。

追加インフラを必要とせず、既存の GitHub ホスト型ワークフローと整合し、ドキュメントの変更が production に反映されたときに自動公開されるホスティング手段が必要である。

## 決定

`docs/` 以下または ワークフロー定義ファイル（`.github/workflows/deploy-docs.yaml`）自体に変更があった `production` ブランチへのプッシュのたびに、`docs/` の内容を GitHub Pages の静的サイトとして公開する。

ワークフローは次のとおり:

1. **build** ジョブ: リポジトリをチェックアウトし、`actions/upload-pages-artifact` を使って `docs/` ディレクトリを GitHub Pages アーティファクトとしてアップロードする。
2. **deploy** ジョブ: `actions/deploy-pages` を使ってアップロードしたアーティファクトを GitHub Pages にデプロイし、生成された URL を `github-pages` 環境に書き込む。

ワークフローは `docs/` 以下またはワークフロー定義自体のパスに触れる `production` プッシュ時にのみトリガーされる。ドキュメントに関係しない `production` へのプッシュでは Pages のデプロイは起動しない。`concurrency` グループ `"pages"` によって同時デプロイを防ぐ。進行中のデプロイはキャンセルしない（`cancel-in-progress: false`）ことで、部分的な公開を回避する。

必要な権限: `contents: read`、`pages: write`、`id-token: write`。

## 影響

### ポジティブな影響

- 外部ホスティングインフラを一切持たず、安定した URL でドキュメントをブラウズできる。
- デプロイパイプラインは 2 ステップと最小限であり、GitHub のホスト型ランナーエコシステム内で完結する。
- パスフィルタリング（`docs/**`）により、ドキュメントに無関係なコード変更によって Pages デプロイが起動されず、Pages 環境を安定に保てる。
- 同時デプロイのキャンセル無効化（`cancel-in-progress: false`）により、部分的な公開によって Pages サイトが破損した状態になることを防ぐ。

### ネガティブな影響

- GitHub Pages は GitHub 固有の機能であり、別のホスティングプラットフォーム（例: Netlify、Cloudflare Pages）に移行する場合はワークフローを置き換える必要がある。
- 公開サイトは `production` ブランチの内容のみを反映する。フィーチャーブランチのドキュメントは自動的にプレビューされない。
- 生の `docs/` ディレクトリのみが配信される。将来 静的サイトジェネレーターを導入する場合は、ワークフローにビルドステップを追加しなければならない。

## 検討した代替案

### 外部静的ホスティング（Netlify、Cloudflare Pages、Vercel）

ブランチプレビューや高機能なビルドパイプラインを提供するが、外部サービスへの依存を導入する。組み込みの GitHub 機能で要件を満たせるときに外部ベンダーに依存することは、[ADR-0001](0001-avoid-lock-in.ja.md) のベンダー中立方針と矛盾するため却下。

### 手動公開

自動化のオーバーヘッドをなくせるが、ドキュメントが古くなるリスクがあり、変更のたびに人間が調整しなければならない。却下。

## 補足

- `.github/workflows/deploy-docs.yaml` がワークフロー定義の全体である。
- 正典ソース原則は ADR-0010 に定められている。
- ソース: `.github/workflows/deploy-docs.yaml`。
