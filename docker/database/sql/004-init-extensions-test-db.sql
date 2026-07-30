-- このファイルはCI環境のテスト用データベース初期化でも使用されます。

\c test
CREATE EXTENSION IF NOT EXISTS pg_trgm;
