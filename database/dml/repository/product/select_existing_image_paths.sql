-- name: ListExistingProductImagePaths :many
-- 与えた画像パスのうち、いずれかの商品が実際に参照しているものを返す。
-- 未参照オブジェクトの回収（product-image-gc）で「消してよいか」を判定する取得元で、
-- ここに現れなかったパスが孤児にあたる。
-- 論理削除された画像は差し替え履歴であって現在の参照ではないため、生存行だけを参照元として数える。
SELECT DISTINCT pi.image_path
FROM product_images AS pi
WHERE pi.image_path = ANY(sqlc.arg('image_paths')::TEXT [])
    AND pi.deleted_at IS NULL;
