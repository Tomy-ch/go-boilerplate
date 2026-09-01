CREATE TABLE IF NOT EXISTS inquiry_messages (
    id UUID NOT NULL,
    inquiry_id UUID NOT NULL,
    author_kind TEXT NOT NULL,
    author_subject_id UUID NOT NULL,
    body TEXT NOT NULL,
    sequence BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT inquiry_messages_id_primary PRIMARY KEY (id),
    CONSTRAINT inquiry_messages_inquiry_id_foreign FOREIGN KEY (inquiry_id) REFERENCES inquiries (id) ON DELETE CASCADE,
    CONSTRAINT inquiry_messages_author_subject_id_foreign FOREIGN KEY (author_subject_id) REFERENCES users (id),
    CONSTRAINT inquiry_messages_author_kind_check CHECK (author_kind IN ('user', 'operator')),
    CONSTRAINT inquiry_messages_sequence_positive CHECK (sequence >= 1),
    -- 採番と同一 tx で書く限り到達しない防御。連続 prefix の不変条件を DB で守る最後の砦。
    CONSTRAINT inquiry_messages_inquiry_id_sequence_unique UNIQUE (inquiry_id, sequence)
);

COMMENT ON TABLE inquiry_messages IS '問い合わせメッセージ';
COMMENT ON COLUMN inquiry_messages.id IS 'ID';
COMMENT ON COLUMN inquiry_messages.inquiry_id IS '所属する問い合わせのID';
COMMENT ON COLUMN inquiry_messages.author_kind IS '送り手の種別（user / operator）';
COMMENT ON COLUMN inquiry_messages.author_subject_id IS '送り手のユーザーID';
COMMENT ON COLUMN inquiry_messages.body IS '本文';
COMMENT ON COLUMN inquiry_messages.sequence IS '問い合わせ内の位置（1 起算）';
COMMENT ON COLUMN inquiry_messages.created_at IS '作成日時';
