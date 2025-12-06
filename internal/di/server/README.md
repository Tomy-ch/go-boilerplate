# server

概要:  
このディレクトリは、**Echo サーバーの初期化・起動・DI 管理を担うサーバーモジュール層**です。  
`ServeModule` を中心に、アプリケーションの HTTP サーバー立ち上げ・ライフサイクル管理を提供し、  
`extension` 下の各拡張（middleware / configurator / observability / security など）と連動して  
**HTTP スタック全体のエントリーポイント**として機能します。

## 役割

- Echo サーバーの生成 (`NewAppServer`)
- HTTP サーバーの起動 (`ServeHTTP`)
- Uber FX による DI 統合（httpstack で構築された拡張を集約）
- サーバー起動・停止のライフサイクル管理
- controller / handler 層が利用する HTTP 基盤を提供

このディレクトリは **「HTTP サーバーそのものを動かすためのコア層」** です。

## 必要度

### 本番運用での必須度

- 必須度: **本番運用で必須**

理由:  

- サーバーの起動はアプリケーションの根幹であるため必須  
- 拡張ミドルウェア（セキュリティ / ロギング / CORS / Observability）が本番で正しく適用されるため  
- fx ライフサイクルと連携し、適切な shutdown / cleanup が保証されるため  

### 開発/テスト運用での必須度

- 必須度: **開発/テスト運用で必須**

理由:  

- ローカル環境でも Echo サーバーを実際に動かし動作確認するため  
- E2E / integration テストで HTTP サーバーが必要  
- 拡張ミドルウェアの動作確認（CORS / エラー変換 / リクエスト整形など）に不可欠  

## 無効化した場合の影響

- HTTP サーバーが起動しなくなる  
- ミドルウェアや拡張機能（requestid, logging, cors, security）の適用が不可能  
- controller / router 層が機能しなくなり、REST API 全体が動作不能  
- fx のライフサイクル管理が正しく行われず、安全な shutdown が保証されない

**実質的にアプリケーションは動作不能となるため、無効化は不可能です。**

## 注意点

- `ServeModule` は必ず DI の最終層として読み込む必要があります  
  → ミドルウェアや configurator を適用した後でサーバーを起動するため  
- `ServeHTTP` はブロッキングするため、fx のライフサイクル外で直接呼び出さないこと  
- `app.NewAppServer` は副作用がある処理であるため、domain/usecase から参照しないこと  
- サーバー設定は `controller/server` 層に寄せ、**domain と infra へ漏れない構成**を保つこと  
- extension 配下の設定（security / inbound / outbound / instrumentation）は  
  **HTTPStackModule → ServeModule の順** で適用される
