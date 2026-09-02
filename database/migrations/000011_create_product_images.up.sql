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

-- 1 商品が保持できる生存画像の枚数に上限を課す。多重防御としての位置づけと、下記のレースがなぜ
-- 到達しないかは docs/spec/product/domain.md の Cross-field Invariants を参照。
-- 値を変えるときは internal/domain/product の maxImages と両方を揃える。
--
-- 件数は行の述語として書けず CHECK では表現できないためトリガで数える。数えるのは生存行だけなので、
-- 置き換えに伴う論理削除のように枚数が下がる更新は素通りする。
--
-- このトリガ単体はレースを閉じない。安全性は書き込み経路の直列化に依存する。
--
-- search_path を固定するのは、非修飾の product_images が探索パス次第で別スキーマの同名テーブルに
-- 解決されうるため（PostgreSQL の関数ハードニングの定石）。
--
-- 以下 2 箇所で抑止している CP03 は組み込み関数の大文字化を課す規則で、ユーザー定義関数の
-- 識別子にも当たってしまう。識別子は他のスキーマ要素と揃えて snake_case を保つ。
CREATE OR REPLACE FUNCTION product_images_assert_max_per_product()  -- noqa: CP03
RETURNS TRIGGER AS $$
DECLARE
    live_count INTEGER;
BEGIN
    -- 論理削除された行は枚数を増やさない。
    IF NEW.deleted_at IS NOT NULL THEN
        RETURN NULL;
    END IF;

    SELECT COUNT(*) INTO live_count
    FROM product_images AS pi
    WHERE pi.product_id = NEW.product_id
        AND pi.deleted_at IS NULL;

    IF live_count > 20 THEN
        RAISE EXCEPTION 'product % holds % live images, which exceeds the maximum of 20',
            NEW.product_id, live_count
            USING ERRCODE = 'check_violation',
                  CONSTRAINT = 'product_images_max_per_product';
    END IF;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql SET search_path = pg_catalog, public;

-- 枚数を増やし得るのは行の追加と、product_id の付け替え・論理削除の取り消しだけなので、その 3 つに絞る。
CREATE TRIGGER product_images_max_per_product
AFTER INSERT OR UPDATE OF product_id, deleted_at ON product_images
FOR EACH ROW
EXECUTE FUNCTION product_images_assert_max_per_product();  -- noqa: CP03

COMMENT ON TABLE product_images IS '商品画像';
COMMENT ON COLUMN product_images.id IS 'ID';
COMMENT ON COLUMN product_images.product_id IS '商品ID';
COMMENT ON COLUMN product_images.image_path IS '画像パス';
COMMENT ON COLUMN product_images.display_sort IS '表示順';
COMMENT ON COLUMN product_images.deleted_at IS '削除日時';
COMMENT ON COLUMN product_images.created_at IS '作成日時';
COMMENT ON COLUMN product_images.updated_at IS '更新日時';
