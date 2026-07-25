-- name: IncrementProductStock :exec
-- 在庫を数量分復元（加算）する。キャンセル時の在庫復元は #571 の防御的減算の逆操作であり、
-- 相対更新（quantity + 数量）のため売り越しを生まず在庫不足ガードは不要（購入行ロック下で実行）。
UPDATE products
SET
    quantity = quantity + @quantity_param,
    updated_at = NOW()
WHERE id = @product_id_param;
