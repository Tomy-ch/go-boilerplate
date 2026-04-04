# コントローラー層（`internal/controller`）ガイド

## オニオンアーキテクチャでの役割

- **外界（HTTP/REST）とアプリケーションの境界面**。
- **プロトコル適応（Adapter）** を担い、入力を **アプリ語彙（DTO/Value）** に変換して **Usecase** を呼ぶ。
- **出力整形（Presenter）** でUsecaseの結果を **OpenAPI のレスポンス型** へ詰め替える。
- 例外（`error`）を **HTTP ステータス** ＋ **エラーコード** へマッピング（`apperror` → Status）。

> ポイント：**ビジネスロジックは一切持たない**。持つのは「HTTPの解釈と整形」だけ。

## サーバーのエントリポイントの実装(`internal/controller/handler`)

サーバーのエントリポイントの実装については、[internal/controller/handler/README.md](handler/README.ja.md)を参照してください。

## ジョブのエントリポイントの実装(`internal/controller/job`)

ジョブのエントリポイントの実装については、[internal/controller/job/README.md](job/README.ja.md)を参照してください。
