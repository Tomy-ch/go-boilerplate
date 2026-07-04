---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [persistence, search]
---

# ADR-0028: GENERATED STORED 列と GIN trgm インデックスを使用したデータベース内全文検索

English canonical: [0028-in-database-full-text-search.md](../../adr/0028-in-database-full-text-search.md)

## ステータス

accepted

## 背景

ユーザー検索 API は、姓・名・メールアドレス・電話番号・市区町村・番地・建物名・郵便番号など複数のユーザーフィールドにわたるキーワードマッチングを要求する。実装戦略はいくつか考えられる。

- **アプリケーション層フィルタリング**: 全行を取得し Go でフィルタリングする。大規模ではあまりにも低速で、インデックスの恩恵がない。
- **個別列への複数 `ILIKE` 条件**: 8 列にまたがる論理和 WHERE 句を含む複雑な SQL を生成する。各列に個別のインデックスパスが必要で、単一インデックスで全列を効率的にカバーできない。
- **外部検索エンジン**（例: Elasticsearch、OpenSearch）: 最高レベルの関連度判定と NLP 機能を持つが、追加のインフラ依存、最終的整合性を持つ同期機構、大きな運用負担を追加する。
- **PostgreSQL `tsvector` / `tsquery`**: ステミングとランキングを備えた言語対応全文検索。強力だが言語辞書の設定が必要で、任意の部分文字列マッチングには向かない。
- **PostgreSQL pg_trgm と計算列**: GIN でインデックス化したトライグラムベースの部分文字列検索。`ILIKE` を効率的にサポートし、言語設定不要で、既存の PostgreSQL インスタンス内のみで完結する。

データセットサイズと現在の検索要件は、外部検索エンジンや NLP グレードの言語処理を必要としない。関連するすべてのフィールドにわたる効率的な部分文字列マッチングで十分である。

## 決定

`users` テーブルに `search_text` という名前の `GENERATED ALWAYS AS (...) STORED` 列を追加する（マイグレーション `000011_users_table_search_text_column.up.sql`）。書き込み時に関連フィールドをスペースで結合する。

```sql
search_text TEXT GENERATED ALWAYS AS (
    COALESCE(first_name, '') || ' '
    || COALESCE(last_name, '') || ' '
    || COALESCE(email, '') || ' '
    || COALESCE(phone, '') || ' '
    || COALESCE(city, '') || ' '
    || COALESCE(street, '') || ' '
    || COALESCE(building, '') || ' '
    || COALESCE(postal_code, '')
) STORED
```

`gin_trgm_ops` を使用した GIN インデックスを `search_text` に作成する。

```sql
CREATE INDEX users_search_text_trgm_idx ON users
USING gin (search_text gin_trgm_ops);
```

キーワード検索クエリは `ILIKE ANY(patterns::TEXT[])` を使って `search_text` と照合する。このクエリは `users`+`prefectures` 集約境界をまたいでビュー固有の DTO を返すため、Repository ではなく QueryService 経由で実行される（[ADR-0025](0025-lightweight-cqrs.ja.md) を参照）。

## 影響

### ポジティブな影響

- 生成列は PostgreSQL が自動的に維持するため、アプリケーションの書き込みで検索フィールドを別途更新する必要がない。
- GIN trgm インデックスによる `ILIKE ANY` は、言語設定なしに典型的なデータセットサイズで効率的な部分文字列マッチングを提供する。
- 追加インフラ不要: この機能は PostgreSQL のみで完結する。
- 単一の `search_text` 列によってクエリが簡潔になる。8 つの論理和条件の代わりに 1 つの `ILIKE ANY` 条件で済む。

### ネガティブな影響

- `search_text` は非正規化された結合列である。新しい検索対象フィールドを追加するには、生成列の式を変更し GIN インデックスを再構築するマイグレーションが必要になる。
- pg_trgm の `ILIKE` には関連度ランキングがなく、すべてのマッチが等しく扱われる。
- pg_trgm はステミング、同義語展開、言語対応トークナイゼーションをサポートしない。
- GIN インデックスは `search_text` のサイズに比例して書き込みオーバーヘッドとストレージを増加させる。

## 検討した代替案

### 個別列への複数 ILIKE 条件

スキーマ変更は不要だが、8 列にまたがる論理和 WHERE を単一インデックスでカバーできず、各列でインデックスのスキャンとフィルタリングが必要になる。非自明な行数での性能上の理由で却下。

### 外部検索エンジン（Elasticsearch など）

最高の関連度判定と NLP 機能を持つ。インフラ依存、最終的整合性の同期レイヤー、運用負担（インデックス管理、クラスター健全性監視）を追加する。現在の要件では複雑さが正当化されないとして、時期尚早と判断し却下。

### PostgreSQL tsvector / tsquery

GIN インデックスを使った言語対応全文検索。ステミングと `ts_rank` 関連度スコアリングをサポートする。日本語コンテンツには辞書設定が必要で、trgm の `ILIKE` ほど自然に任意の部分文字列マッチングをサポートしない。部分文字列マッチングが主なユースケースであり、trgm が言語設定不要であるため却下。

## 補足

- 出典: [`database/migrations/000011_users_table_search_text_column.up.sql`](../../../database/migrations/000011_users_table_search_text_column.up.sql)。
- 出典: [`database/dml/query_service/user/select_users_by_keyword.sql`](../../../database/dml/query_service/user/select_users_by_keyword.sql)。
- 関連: [ADR-0025](0025-lightweight-cqrs.ja.md)（この検索は集約境界をまたぐため QueryService を使用）。
