-- このファイルはCI環境のテスト用データベース初期化でも使用されます。

\c test
ALTER DATABASE test SET timezone TO 'Asia/Tokyo';
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION IF NOT EXISTS hstore;
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;
