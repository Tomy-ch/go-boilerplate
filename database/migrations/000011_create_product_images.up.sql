CREATE TABLE IF NOT EXISTS product_images (
    id UUID NOT NULL,
    product_id UUID NOT NULL,
    image_path TEXT NOT NULL,
    display_sort SMALLINT NOT NULL,
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT product_images_id_primary PRIMARY KEY (id),
    CONSTRAINT product_images_product_id_foreign FOREIGN KEY (product_id) REFERENCES products (id)
);

-- 一意性は生存行にだけ課す。UNIQUE (product_id, display_sort, deleted_at) では NULL 同士が相異なる値として
-- 扱われるため、deleted_at IS NULL の行が同じ (product_id, display_sort) で並存でき、一意性が消える。
CREATE UNIQUE INDEX product_images_product_id_display_sort_unique ON product_images (
    product_id, display_sort
)
WHERE deleted_at IS NULL;

COMMENT ON TABLE product_images IS '商品画像';
COMMENT ON COLUMN product_images.id IS 'ID';
COMMENT ON COLUMN product_images.product_id IS '商品ID';
COMMENT ON COLUMN product_images.image_path IS '画像パス';
COMMENT ON COLUMN product_images.display_sort IS '表示順';
COMMENT ON COLUMN product_images.deleted_at IS '削除日時';
COMMENT ON COLUMN product_images.created_at IS '作成日時';
COMMENT ON COLUMN product_images.updated_at IS '更新日時';
