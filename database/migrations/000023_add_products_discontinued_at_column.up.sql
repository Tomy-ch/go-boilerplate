ALTER TABLE products ADD COLUMN IF NOT EXISTS discontinued_at TIMESTAMPTZ;

COMMENT ON COLUMN products.discontinued_at IS '廃番日時';

-- 廃番の商品だけを引く絞り込みが走査する。廃番は商品全体のごく一部に留まる想定のため部分索引とし、
-- 廃番でない大多数の行を索引に載せない。
CREATE INDEX IF NOT EXISTS products_discontinued_at_idx ON products (discontinued_at)
WHERE discontinued_at IS NOT NULL;
