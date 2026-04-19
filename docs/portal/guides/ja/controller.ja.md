# コントローラー層（`internal/controller`）ガイド

[English](README.md) | 日本語

## オニオンアーキテクチャでの役割

- **外界（HTTP/CLI）とアプリケーションの境界面**
- **プロトコル適応（Adapter）** を担い、入力を **アプリ語彙（DTO/Value）** に変換して **Usecase** を呼ぶ
- **出力整形（Presenter）** で Usecase の結果を **OpenAPI のレスポンス型** へ詰め替える
- 例外（`error`）を **HTTP ステータス** + **エラーコード** へマッピング（`apperror` → Status）

> ポイント：**ビジネスロジックは一切持たない**。持つのは「HTTP / CLI の解釈と整形」だけ。

## ディレクトリ構成

```text
internal/controller/
├── handler/        # HTTP ハンドラ（サーバーのエントリポイント）
├── job/            # ジョブコントローラ（CLI のエントリポイント）
├── server/         # Echo インスタンス生成・サーバー起動
├── httpstack/      # HTTP ミドルウェア群
├── error/response/ # エラーレスポンス生成
└── ctxhelper/      # Echo コンテキストヘルパー
```

## サブディレクトリの役割

|ディレクトリ|説明|詳細|
|---|---|---|
|`handler/`|HTTP リクエストを受け取り Usecase へ委譲するハンドラ|[README](handler/README.ja.md)|
|`job/`|CLI から起動されるジョブのコントローラ|[README](job/README.ja.md)|
|`server/`|Echo インスタンスの初期化と DI ライフサイクルへの統合|[README](server/README.ja.md)|
|`httpstack/`|ミドルウェア群（CORS, セキュリティ, ログ, 認証, レートリミット等）|[README](httpstack/README.ja.md)|
|`error/response/`|統一的な HTTP エラーレスポンスの生成と apperror マッピング|[README](error/response/README.ja.md)|
|`ctxhelper/`|Echo コンテキストへの値の設定・取得ヘルパー|[README](ctxhelper/README.ja.md)|

## 依存関係ルール

```mermaid
flowchart TB
    Controller --> Usecase
    Controller --> apperror
    Controller --> Presenter

    Controller -. forbidden .-> Domain
    Controller -. forbidden .-> Infrastructure
    Controller -. forbidden .-> Database
```

Controller は **Usecase を通してのみ下位層にアクセス**します。
