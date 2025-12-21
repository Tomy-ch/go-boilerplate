ALTER TABLE users ADD COLUMN search_text TEXT
GENERATED ALWAYS AS (
    COALESCE(first_name, '') || ' '
    || COALESCE(last_name, '') || ' '
    || COALESCE(email, '') || ' '
    || COALESCE(phone, '') || ' '
    || COALESCE(city, '') || ' '
    || COALESCE(street, '') || ' '
    || COALESCE(building, '') || ' '
    || COALESCE(postal_code, '')
) STORED;

COMMENT ON COLUMN users.search_text IS '全文検索用テキスト';

CREATE INDEX users_search_text_trgm_idx ON users
USING gin (search_text gin_trgm_ops);
