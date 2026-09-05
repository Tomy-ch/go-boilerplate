CREATE TABLE IF NOT EXISTS coupons (
    id UUID NOT NULL,
    user_id UUID NOT NULL,
    discount_kind SMALLINT NOT NULL,
    discount_value NUMERIC NOT NULL,
    scope_kind SMALLINT NOT NULL,
    scope_target_id UUID,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    issued_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT coupons_id_primary PRIMARY KEY (id),
    CONSTRAINT coupons_user_id_foreign FOREIGN KEY (user_id) REFERENCES users (id)
);

COMMENT ON TABLE coupons IS 'クーポン';
COMMENT ON COLUMN coupons.id IS 'ID';
COMMENT ON COLUMN coupons.user_id IS '受給者のユーザーID';
COMMENT ON COLUMN coupons.discount_kind IS '値引き種別コード';
COMMENT ON COLUMN coupons.discount_value IS '値引きの値（定額なら金額、定率なら率）';
COMMENT ON COLUMN coupons.scope_kind IS '適用範囲種別コード';
COMMENT ON COLUMN coupons.scope_target_id IS '適用範囲の対象ID（カテゴリIDまたは商品ID。全体のときNULL）';
COMMENT ON COLUMN coupons.expires_at IS '有効期限';
COMMENT ON COLUMN coupons.used_at IS '使用日時';
COMMENT ON COLUMN coupons.issued_at IS '発行日時';
COMMENT ON COLUMN coupons.created_at IS '作成日時';
COMMENT ON COLUMN coupons.updated_at IS '更新日時';

-- 保有クーポンの一覧が引く。受給者は発行時に確定し以後移らないため、user_id 単独で足りる。
CREATE INDEX coupons_user_id_idx ON coupons (user_id);

-- 廃番を再実行したときに、その商品を根拠に発行済みの枚数を数える経路が引く。
-- 対象を持たない適用範囲（全体）は数えないため、部分索引で NULL の行を載せない。
CREATE INDEX coupons_scope_target_id_idx ON coupons (scope_target_id)
WHERE scope_target_id IS NOT NULL;
