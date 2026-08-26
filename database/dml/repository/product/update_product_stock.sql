-- name: UpdateProductStock :one
-- 在庫数を更新し、採番後のバージョンを返します。
-- lock_version の加算は SQL 側で行う（採番の権威の置き場所は docs/spec/product/domain.md の
-- Product.Update を参照）。
-- 在庫更新でもバージョンを進めることで、更新前のバージョンを条件とする部分更新（UpdateProduct）が
-- 在庫の変化を上書きせずに 0 行で弾かれます。
-- WHERE の lock_version 一致は、行ロックを取らずに呼ばれた場合に備える二重防御で、
-- 該当行なし（0 行）は呼び出し側が衝突として扱います。
UPDATE products
SET
    quantity = sqlc.arg('quantity'),
    lock_version = products.lock_version + 1,
    updated_at = NOW()
WHERE products.id = sqlc.arg('id')
    AND products.lock_version = sqlc.arg('current_version')
RETURNING products.lock_version;
