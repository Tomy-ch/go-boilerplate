-- name: MarkOutboxFailed :execrows
-- publish 失敗時に last_error を記録し、次に claim してよい時刻をバックオフ後の時刻へ進める。
-- attempts は診断のために加算し続けるが、dead 判定の基準ではない（判定はエラー分類。ADR-0058）。
UPDATE outbox
SET
    attempts = attempts + 1,
    last_error = $2,
    next_attempt_at = $3
WHERE id = $1
    AND status = 'pending';
