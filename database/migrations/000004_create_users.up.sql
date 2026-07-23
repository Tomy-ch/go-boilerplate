CREATE TABLE IF NOT EXISTS users (
    id UUID NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    email VARCHAR(100) NOT NULL,
    phone VARCHAR(20) NOT NULL,
    prefecture_id UUID NOT NULL,
    city VARCHAR(100) NOT NULL,
    street VARCHAR(255) NOT NULL,
    building VARCHAR(255),
    postal_code VARCHAR(8) NOT NULL,
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT users_id_primary PRIMARY KEY (id),
    CONSTRAINT users_email_unique UNIQUE (email),
    CONSTRAINT users_prefecture_id_foreign FOREIGN KEY (prefecture_id) REFERENCES prefectures (id)
);

COMMENT ON TABLE users IS 'ユーザ';
COMMENT ON COLUMN users.id IS 'ID';
COMMENT ON COLUMN users.first_name IS '名前';
COMMENT ON COLUMN users.last_name IS '苗字';
COMMENT ON COLUMN users.email IS 'メールアドレス';
COMMENT ON COLUMN users.phone IS '電話番号';
COMMENT ON COLUMN users.prefecture_id IS '都道府県ID';
COMMENT ON COLUMN users.city IS '市区町村';
COMMENT ON COLUMN users.street IS '番地';
COMMENT ON COLUMN users.building IS '建物名';
COMMENT ON COLUMN users.postal_code IS '郵便番号';
COMMENT ON COLUMN users.deleted_at IS '削除日時';
COMMENT ON COLUMN users.created_at IS '作成日時';
COMMENT ON COLUMN users.updated_at IS '更新日時';
