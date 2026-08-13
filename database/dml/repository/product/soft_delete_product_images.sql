-- name: SoftDeleteProductImages :exec
-- 商品が現在参照している画像をまとめて論理削除する。既に論理削除済みの行は削除日時を上書きしない。
-- 生存行が無い商品に対しては 0 行更新となり、成功として返る。
UPDATE product_images
SET
    deleted_at = NOW(),
    updated_at = NOW()
WHERE product_images.product_id = sqlc.arg('product_id')
    AND product_images.deleted_at IS NULL;
