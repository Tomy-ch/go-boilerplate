-- name: ListInquiriesForOperatorFirst :many
-- 運営向けに問い合わせを (updated_at DESC, id DESC) の安定順で先頭ページ取得する。
-- 本文は含めない（一覧は問い合わせの行だけで組み立てる。最新メッセージの要約は feed の event が運ぶ）。
SELECT sqlc.embed(i)
FROM inquiries AS i
ORDER BY i.updated_at DESC, i.id DESC
LIMIT sqlc.arg('page_size');

-- name: ListInquiriesForOperatorAfter :many
-- 同上を cursor（updated_at, id）より後ろから取得する。複合 cursor の比較は行値式で行い、
-- updated_at が同値の行を id で決着させる（先頭ページと同じ全順序を保つ）。
SELECT sqlc.embed(i)
FROM inquiries AS i
-- 2 番目の要素だけ明示的にキャストする。キャストが無いと sqlc は行値式の 2 要素目の型を推論できず、
-- 1 要素目（timestamptz）を流用した引数を生成する。1 要素目にもキャストを付けると
-- sqlc.yaml の pg_catalog.timestamptz 上書きから外れ pgtype へ落ちるため、こちらは付けない。
WHERE (i.updated_at, i.id) < (sqlc.arg('cursor_updated_at'), sqlc.arg('cursor_id')::UUID)
ORDER BY i.updated_at DESC, i.id DESC
LIMIT sqlc.arg('page_size');
