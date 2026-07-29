CREATE TABLE IF NOT EXISTS purchases (
    id UUID NOT NULL,
    code VARCHAR(50) NOT NULL,
    user_id UUID NOT NULL,
    status_id UUID NOT NULL,
    subtotal_amount BIGINT NOT NULL,
    tax_amount BIGINT NOT NULL,
    shipping_fee BIGINT NOT NULL,
    total_amount BIGINT NOT NULL,
    ordered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    paid_at TIMESTAMPTZ,
    canceled_at TIMESTAMPTZ,
    shipped_at TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT purchases_id_primary PRIMARY KEY (id),
    CONSTRAINT purchases_code_unique UNIQUE (code),
    CONSTRAINT purchases_user_id_foreign FOREIGN KEY (user_id) REFERENCES users (id),
    CONSTRAINT purchases_status_id_foreign FOREIGN KEY (status_id) REFERENCES purchase_statuses (id)
);

COMMENT ON TABLE purchases IS '購入';
COMMENT ON COLUMN purchases.id IS 'ID';
COMMENT ON COLUMN purchases.code IS 'コード';
COMMENT ON COLUMN purchases.user_id IS 'ユーザID';
COMMENT ON COLUMN purchases.status_id IS '購入ステータスID';
COMMENT ON COLUMN purchases.subtotal_amount IS '小計金額';
COMMENT ON COLUMN purchases.tax_amount IS '税金額';
COMMENT ON COLUMN purchases.shipping_fee IS '配送料';
COMMENT ON COLUMN purchases.total_amount IS '合計金額';
COMMENT ON COLUMN purchases.ordered_at IS '注文日時';
COMMENT ON COLUMN purchases.paid_at IS '支払日時';
COMMENT ON COLUMN purchases.canceled_at IS 'キャンセル日時';
COMMENT ON COLUMN purchases.shipped_at IS '発送日時';
COMMENT ON COLUMN purchases.delivered_at IS '配達日時';
COMMENT ON COLUMN purchases.created_at IS '作成日時';
COMMENT ON COLUMN purchases.updated_at IS '更新日時';

-- 購入履歴一覧（GET /v1/purchases）のユーザー別 keyset ページネーション用複合インデックス。
-- WHERE user_id = $1 ORDER BY ordered_at DESC, id DESC を index range scan で処理する
-- （FK 制約は参照列に索引を張らないため明示的に追加する）。
CREATE INDEX purchases_user_id_ordered_at_id_idx
ON purchases (user_id, ordered_at DESC, id DESC);
