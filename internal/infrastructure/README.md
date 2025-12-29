# インフラ層（`internal/infrastructure`）ガイド

## オニオンアーキテクチャでの役割

- **外部技術（DB・外部API・メッセージング等）へのアクセス実装**を担う層。
- **Domain が定義した Repository インターフェース**を満たす具体実装を提供する（依存関係の逆転：Domainは抽象のみ、Infraが具体）。
- I/O・接続・ドライバ・リトライ・ロギングなど**技術的な詳細**をここに閉じ込め、上位層に漏らさない。

## RDBアクセス実装（`internal/infrastructure/rdb`）

RDBアクセス実装については、[internal/infrastructure/rdb/README.md](rdb/README.md)を参照してください。
