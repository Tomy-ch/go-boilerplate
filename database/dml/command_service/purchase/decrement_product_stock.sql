-- name: DecrementProductStock :execrows
-- 在庫を数量分減算する。防御的に quantity >= 減算数を併用し、売り越しをアトミックに弾く（更新 0 行なら在庫不足）。
-- ロック取得後に検証済みのため通常は 0 行にならないが、fail-closed の二重防御として残す（ADR-0100）。
-- この条件は domain の売り越し判定（purchase.New が返す ErrInsufficientStock）を言い換えたもので、
-- 独立した規則ではない。判定が変わったらこちらも追随させること（逆は無い。ADR-0027 § Derivation）。
UPDATE products
SET
    quantity = quantity - @quantity_param,
    updated_at = NOW()
WHERE id = @product_id_param
    AND quantity >= @quantity_param;
