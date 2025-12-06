# inbound

概要: `inbound` は **HTTP リクエスト（入力）側の前処理を行うミドルウェアおよびサーバー設定を DI 経由で提供するレイヤー** です。
リクエストの受信時に実行される Binder / Validator / URI 正規化 / IP 抽出 などを統一的に管理し、API 入口の品質・安全性・一貫性を保証します。

## 役割

このディレクトリには以下のような **“入力整形（Input Processing）” を行う拡張機能**が含まれます。

### 1. **Binder**

- Echo の Bind 処理をアプリケーション仕様に合わせて拡張
- JSON / Query / Form の入力変換の共通処理を提供
- 実質的に「すべてのハンドラの前段に必ず動く」基本コンポーネント

### 2. **URI 正規化ミドルウェア**

- URI の末尾スラッシュ削除
- 正規化されたパスでハンドラに到達させる
- Priority を持って PreMiddleware として適用

### 3. **IP Extractor**

- クライアント IP の抽出
- Proxy や X-Forwarded-For 対応
- SecurityConfig に従って扱うヘッダを制御

### 4. **Validator（OpenAPI バリデーション）**

- OpenAPI 仕様に基づいた **リクエスト自動バリデーション**
- 型・format・required・enum などの検証
- controller 層にロジックが漏れないようにするための重要機能

## 必要度

### 本番運用での必須度

- 必須度: **本番運用で必須**

理由:

- 不正リクエストをサーバー内部に到達させないため
- OpenAPI ベースのバリデーションにより API の仕様逸脱を自動検出
- クライアント IP の取得はロギング・セキュリティ上必須
- URI 正規化がないとルーティングがブレる可能性がある
- 期待しない入力が domain/usecase に流れ込まないようにするため

### 開発/テスト運用での必須度

- 必須度: **開発/テスト運用で推奨**

理由:

- ローカルでも API リクエストの仕様逸脱を早期検出できる
- IP 抽出や URI 正規化を本番と同じ環境で評価できる
- Binder の挙動（JSON パース等）を環境差異なくテスト可能
- Handler 側のテスト作成が容易になる（前処理を DI に任せられる）

## 無効化した場合の影響

- 不正な入力が素通りし、controller/usecase 層が壊れやすくなる
- API 仕様と実装の不整合を検出できない
- ルーティング揺れ（/users と /users/ の違い）によるバグ発生
- クライアント IP 情報が失われ、ログ相関やセキュリティ分析が困難
- リクエストごとの整形処理がハンドラ側に漏れ、保守性が低下

## 注意点

- **Validator（OpenAPI バリデーション）は UseMiddleware、URI は PreMiddleware**
  → Priority により適切な順序で動くように設計されている
- IP Extractor は SecurityConfig / ApplicationConfig に依存するため
  **本番環境とローカルで挙動が異なる可能性に注意**
- Binder / Validator は handler に影響するため、
  **controller 層・domain 層にロジックを漏らさない**こと
- 新しい inbound 機能を追加する際は、
  **ServeCfg（Echo 設定）か Pre/UseMiddleware のどちらかに分類**すること
