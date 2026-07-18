-- public スキーマの全テーブルを削除する（拡張・関数・型は残す）。
-- db-init（migrate-down → up）は down マイグレーションを要するため、サンプル削除等で
-- down ファイルが失われた「ダーティなスキーマ」には使えない。その代替として、public の
-- 全テーブル（schema_migrations 含む）を CASCADE で drop し、続く migrate-up がディスク上の
-- マイグレーションを最初から再適用できるクリーンな状態へ戻す保守用スクリプト。
-- 本リポジトリのマイグレーションは table のみを作る（型・関数・拡張は作らない）ため、
-- テーブル削除だけで十分。拡張（pg_trgm 等・init SQL 由来）は温存される。
DO $$
DECLARE
  r RECORD;
BEGIN
  FOR r IN SELECT tablename FROM pg_tables WHERE schemaname = 'public' LOOP
    EXECUTE format('DROP TABLE IF EXISTS public.%I CASCADE', r.tablename);
  END LOOP;
END $$;
