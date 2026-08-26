-- name: CreateProductImage :exec
-- 商品画像を 1 件登録する。生存行の (product_id, display_sort) は部分 UNIQUE インデックスが一意に保つため、
-- 同一商品の同じ表示順へ二重に登録すると 23505 で失敗する。
INSERT INTO product_images (
    id,
    product_id,
    image_path,
    display_sort
) VALUES
(
    sqlc.arg('id'),
    sqlc.arg('product_id'),
    sqlc.arg('image_path'),
    sqlc.arg('display_sort')
);
