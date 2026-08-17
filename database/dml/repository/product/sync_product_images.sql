-- 商品画像を、商品が保持する集合へ一致させる 2 本。実行順は呼び出し側（syncImages）が決める。

-- name: SoftDeleteProductImagesNotIn :exec
-- 商品が現在参照している画像のうち、指定した ID の集合に含まれないものを論理削除する。
-- 既に論理削除済みの行は削除日時を上書きしない。空の集合を渡した場合は生存行をすべて論理削除する。
UPDATE product_images
SET
    deleted_at = NOW(),
    updated_at = NOW()
WHERE product_images.product_id = sqlc.arg('product_id')
    AND product_images.deleted_at IS NULL
    AND NOT (product_images.id = ANY(sqlc.arg('image_ids')::UUID []));

-- name: CreateProductImagesIfAbsent :exec
-- 商品画像をまとめて登録する。同じ ID が既にある場合は何もしない。
-- 衝突判定を主キーに限定しているため、生存行の (product_id, display_sort) の重複は従来どおり 23505 で失敗する。
INSERT INTO product_images (
    id,
    product_id,
    image_path,
    display_sort
)
SELECT
    src.id,
    sqlc.arg('product_id'),
    src.image_path,
    src.display_sort
FROM (
    SELECT
        UNNEST(sqlc.arg('ids')::UUID []) AS id,
        UNNEST(sqlc.arg('image_paths')::TEXT []) AS image_path,
        UNNEST(sqlc.arg('display_sorts')::SMALLINT []) AS display_sort
) AS src
ON CONFLICT ON CONSTRAINT product_images_id_primary DO NOTHING;
