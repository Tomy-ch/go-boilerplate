DROP INDEX IF EXISTS users_search_text_trgm_idx;

ALTER TABLE users
DROP COLUMN IF EXISTS search_text;
