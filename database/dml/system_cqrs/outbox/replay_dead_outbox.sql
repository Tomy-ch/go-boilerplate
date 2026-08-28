-- name: ReplayDeadOutbox :execrows
-- dead 行を pending へ戻し再 publish 対象に復帰させる（運用 replay）。
-- $1 が NULL の場合は全 dead 行、指定時は当該 message_id のみを対象とする。
-- next_attempt_at を現在時刻へ戻さないと、dead 化前のバックオフ済み時刻が残って直後に claim されない。
UPDATE outbox
SET
    status = 'pending',
    attempts = 0,
    last_error = NULL,
    next_attempt_at = NOW()
WHERE status = 'dead'
    AND (sqlc.narg('message_id')::UUID IS NULL OR message_id = sqlc.narg('message_id')::UUID);
