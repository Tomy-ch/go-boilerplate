-- name: ExistsInProgressPurchaseByUserID :one
-- 指定ユーザーに進行中の購入が 1 件でも存在するかを返す。進行中は終端ステータス（引数）の否定で
-- 判定するため、ステータスが増えた場合は既定で進行中側に倒れる。ステータスは購入ステータスマスタとの
-- 結合で解決する（購入集約に属する固定参照マスタへの一意な等結合であり、単一集約の read）。
-- 終端コードは seed UUID を焼き込まずドメイン定数から引数で受け取る。
SELECT EXISTS(
    SELECT 1
    FROM purchases AS p
    INNER JOIN purchase_statuses AS ps ON p.status_id = ps.id
    WHERE p.user_id = sqlc.arg('user_id')
        AND NOT (ps.code = ANY(@terminal_status_codes::SMALLINT []))
);
