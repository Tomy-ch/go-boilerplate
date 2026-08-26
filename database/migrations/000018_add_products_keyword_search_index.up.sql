-- 商品名・説明への keyword 検索は ILIKE '%...%' で前方一致ではないため、B-tree では引けない。
-- users の全文検索と同じく trigram 索引で引く（000015_add_users_table_search_text_column）。
-- 一覧は LIMIT で打ち切れるが一致件数は打ち切れないため、無索引のままだと後者だけが常に全走査になる。
-- name と description を別々の索引にするのは、DML の OR 条件をそのまま BitmapOr で引くためで、
-- users のように連結列へ寄せると、キーワードが列の境界をまたいで一致するようになり意味が変わる。
CREATE INDEX products_name_trgm_idx ON products
USING gin (name gin_trgm_ops);

CREATE INDEX products_description_trgm_idx ON products
USING gin (description gin_trgm_ops);
