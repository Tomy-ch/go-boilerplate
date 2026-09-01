CREATE TABLE IF NOT EXISTS inquiries (
    id UUID NOT NULL,
    user_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT inquiries_id_primary PRIMARY KEY (id),
    CONSTRAINT inquiries_user_id_foreign FOREIGN KEY (user_id) REFERENCES users (id),
    CONSTRAINT inquiries_user_id_unique UNIQUE (user_id)
);

-- 運営一覧の keyset ページネーション（updated_at desc, id desc）が引く。
CREATE INDEX inquiries_updated_at_id_index ON inquiries (updated_at DESC, id DESC);

COMMENT ON TABLE inquiries IS '問い合わせ';
COMMENT ON COLUMN inquiries.id IS 'ID';
COMMENT ON COLUMN inquiries.user_id IS '問い合わせを開始した利用者のユーザーID';
COMMENT ON COLUMN inquiries.created_at IS '作成日時';
COMMENT ON COLUMN inquiries.updated_at IS '最後にメッセージが追加された日時';
