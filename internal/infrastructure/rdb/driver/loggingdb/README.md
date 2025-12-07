# loggingdb

概要: **DB へのアクセスに対して SQL 実行ログを付与するためのラッパーレイヤー。実際のクエリ処理は DatabaseDriver に委譲し、ログ整形・出力を追加する。**

## 役割

このディレクトリは、以下のような **SQL ロギングの責務** を担います。

- `DatabaseDriver`（実 DB）をラップし、SQL 実行前後にログを出力する
- 実行クエリ・引数・所要時間・エラー・トレース情報（TraceID/SpanID） をログ構造化フィールドとして出力
- `DBProvider` によるロギング付き DBTX の生成 (`NewLoggingDB`)
- `logger` と `LogFieldBuilder` を注入しログ形式の一貫性を保持
- OpenTelemetry トレース情報と連携するための `observability.ExtractSpan` の利用

上位レイヤ（repository / sqlc / usecase）はログ実装を一切意識せずに SQL ログを取得できます。

## 必要度

### 本番運用での必須度

- 必須度: **本番運用で推奨**

理由:  
DB ログは問題発生時の調査に不可欠です。  
特に以下の場面で強力です。

- N+1 / 遅延クエリの発見
- API 処理単位のトレーシング連動
- エラー SQL の特定

ただし、**ログコストを気にする環境（高トラフィック）では OFF にする構成も許容**されます。  
そのため「必須」ではなく「推奨」。

### 開発/テスト運用での必須度

- 必須度: **開発/テスト運用で必須**

理由:  
開発中は以下の理由で SQL ログが必須級です。

- クエリが正しく発行されているかの確認
- sqlc 生成クエリの挙動把握
- DB テスト時にエラー箇所を迅速に把握

ユニットテスト ～ E2E テストの全範囲で役立ちます。

### 無効化した場合の影響

- SQL ログがすべて出力されなくなる
- 遅いクエリ・誤ったクエリの発見が難しくなる
- OpenTelemetry と SQL レイヤーの紐づきが消える
- repository/usecase 側では動作に影響しない（あくまで観測性の低下）

## 注意点

- loggingdb は **実 DB への I/O を行わない**。あくまで logging wrapper である
- context による TraceID/SpanID を利用するため、handler 層で context を破壊しないこと
- `LogFieldBuilder` に依存しているため、ログ出力形式を変更する場合は builder の設定が必要
- `ExecContext` / `QueryContext` / `QueryRowContext` など、db 操作ごとにログが発生するため大量クエリ処理ではログ量が増える可能性あり
- slow query log を追加したい場合はこのレイヤーに拡張する
