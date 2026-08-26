
# GitHub Pages

本プロジェクトでは、ドキュメントを GitHub Pages で公開します。  
デプロイは GitHub Actions 経由で実行します。

## 概要

本リポジトリの GitHub Pages は、以下の方針で運用します。

- 公開方法: GitHub Actions
- Repository 設定: `Settings > Pages > Source = GitHub Actions`
- デプロイ定義: `.github/workflows/deploy-docs.yaml`
- 実際のデプロイ挙動の正本: GitHub Actions の Workflow

> [!NOTE]
> このドキュメントは GitHub Pages の運用方針と確認ポイントを説明するものです。  
> 実際のビルド手順・デプロイ手順は `.github/workflows/deploy-docs.yaml` を正本として扱ってください。

## GitHub Actions を利用する理由

本リポジトリでは、従来の "Deploy from a branch" による Pages 公開は採用しません。  
GitHub Actions 経由でビルド・デプロイすることで、公開フローを再現可能かつレビュー可能な状態で管理します。

利点:

- デプロイ手順をバージョン管理できる
- ビルド入力を Workflow ファイル上で追跡できる
- ローカルのファイル配置や Repository 設定との差異が起きにくい
- 将来的な拡張を追加しやすい

## Repository 設定

トラブルシューティング前に、以下の Repository 設定を確認してください。

### Settings > Pages

- Source: `GitHub Actions`

ここが正しく設定されていないと、Workflow が成功しても期待通りに公開されない場合があります。

## Workflow

GitHub Pages のデプロイ Workflow は以下です。

```text
.github/workflows/deploy-docs.yaml
```

トリガー条件、ビルド手順、artifact の配置先、deploy 手順などの詳細は、必ず Workflow ファイルを確認してください。

## ベースパス

Project Repository の GitHub Pages は、リポジトリ名を含むパスで公開されます。

```text
https://<username>.github.io/<repository-name>/
```

例:

```text
https://example-org.github.io/example-api/
```

そのため、ドキュメント内のリンクや静的アセット参照には注意が必要です。

### 推奨

- 可能な限り相対パスを使う
- リポジトリ名を含むベースパスでも動作するようにリンクと静的アセット参照を組む

### 非推奨

- `/` をサイトルートとみなす絶対パス

例:

```html
<!-- NG -->
<link rel="stylesheet" href="/styles.css">

<!-- OK -->
<link rel="stylesheet" href="./styles.css">
```

## 注意点

### 正本は Workflow

このドキュメントと Workflow の内容に差異がある場合は、`.github/workflows/deploy-docs.yaml` を正本として扱ってください。

### 静的サイト前提

GitHub Pages は静的ファイル配信です。  
実行時のサーバ処理を前提としたページは、そのままでは動作しません。

### SPA ルーティング

Single Page Application を GitHub Pages に載せる場合、深いパスでリロードすると `404` になることがあります。  
SPA ルーティングを導入する場合は、`404.html` リダイレクトなどのフォールバック戦略を追加してください。

### キャッシュ

GitHub Pages はデプロイ直後でもキャッシュにより旧内容が表示されることがあります。  
更新が見えない場合は、まず Workflow の結果を確認したうえでハードリロードしてください。

## トラブルシューティング

### 更新されない

以下を順番に確認してください。

1. `.github/workflows/deploy-docs.yaml` が起動しているか
2. Workflow が成功しているか
3. `Settings > Pages > Source` が `GitHub Actions` になっているか
4. 公開パスとアセット参照がリポジトリのベースパスに対応しているか

### アセットやリンクが壊れる

主な原因:

- `/assets/...` のようなルート相対パスを使っている
- ドメイン直下公開を前提にしている
- 生成物のパスと GitHub Pages の公開パスが一致していない

### デプロイ挙動が分からない

Workflow ファイルを開き、以下を確認してください。

- trigger 条件
- build コマンド
- 出力ディレクトリまたは upload する artifact
- Pages への deploy ステップ

## 関連ファイル

- `.github/workflows/deploy-docs.yaml`
- `docs/`
- `docs/portal/`

## 運用ルール

ドキュメント公開方法を変更する場合は、以下を必ずセットで更新してください。

1. `.github/workflows/deploy-docs.yaml`
2. このドキュメント

これにより、実際のデプロイ挙動と説明文のズレを防ぎます。
