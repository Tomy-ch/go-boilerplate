-- name: ListExistingProductImagePaths :many
-- 与えた画像パスのうち、いずれかの商品が実際に参照しているものを返す。
-- 未参照オブジェクトの回収（product-image-gc）で「消してよいか」を判定する取得元で、
-- ここに現れなかったパスが孤児にあたる。商品は論理削除を持たないため、生存行だけが参照元になる。
SELECT DISTINCT image_path
FROM products
WHERE image_path = ANY(sqlc.arg('image_paths')::TEXT []);
