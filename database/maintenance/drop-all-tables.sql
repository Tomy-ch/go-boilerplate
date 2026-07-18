-- public スキーマの全テーブル（schema_migrations 含む）を CASCADE で drop する保守用スクリプト。
-- down マイグレーションを失った「ダーティなスキーマ」を、migrate-up で再構築できるクリーンな状態へ戻す。
-- 拡張・関数・型は残す（本リポジトリのマイグレーションは table のみを作るため）。
DO $$
DECLARE
  r RECORD;
BEGIN
  FOR r IN SELECT tablename FROM pg_tables WHERE schemaname = 'public' LOOP
    EXECUTE format('DROP TABLE IF EXISTS public.%I CASCADE', r.tablename);
  END LOOP;
END $$;
