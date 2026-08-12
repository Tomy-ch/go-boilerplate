-- name: IncrementProductStock :execrows
-- 在庫を数量分復元（加算）する。相対更新（quantity + 数量）のため売り越しを生まず在庫不足ガードは不要。
-- 対象行が不存在の場合は影響 0 行として呼び出し側で NotFound へ fail-closed 検出する。
UPDATE products
SET
    quantity = quantity + @quantity_param,
    updated_at = NOW()
WHERE id = @product_id_param;
