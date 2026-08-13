-- name: DecrementProductStock :execrows
-- 在庫を数量分減算する。防御的に quantity >= 減算数を併用し、売り越しをアトミックに弾く（更新 0 行なら在庫不足）。
-- この条件は domain の売り越し判定を言い換えた実行形で、独立した規則ではない。判定が変わったら
-- こちらも追随させること（逆は無い。internal/infrastructure/rdb/README.md の command_service 節）。
UPDATE products
SET
    quantity = quantity - @quantity_param,
    updated_at = NOW()
WHERE id = @product_id_param
    AND quantity >= @quantity_param;
