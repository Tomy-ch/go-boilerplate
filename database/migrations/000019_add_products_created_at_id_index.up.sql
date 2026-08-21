-- 未公開商品を含む一覧（includeUnpublished）は ORDER BY created_at DESC, id DESC で走査する。
-- 既定の一覧が使う products_published_at_id_idx は WHERE published_at IS NOT NULL の部分索引であり、
-- 未公開行を含む走査では引けない。
-- 未公開行は published_at が NULL なので公開日時を並び順の第 1 キーにできない。登録日時は NOT NULL で、
-- 未公開商品にも「いつ登録されたか」という意味が通る唯一の時系列キーであるため、これを軸に採る。
-- 逆順の走査で昇順（sort=publishedAt の昇順指定）も同じ索引で賄えるため、並び順ごとには分けない。
-- price / quantity の範囲条件は INCLUDE 列から評価し、絞り込みで捨てる行の heap 参照を避ける
-- （000010_create_products の公開一覧用索引と同じ意図）。
CREATE INDEX products_created_at_id_idx
ON products (created_at DESC, id DESC)
INCLUDE (price, quantity);
