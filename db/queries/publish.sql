-- name: GetPublishResultByKey :one
SELECT id, channel_id, post_id, idempotency_key, platform_post_id, raw_response, published_at
FROM publish_results
WHERE idempotency_key = $1;

-- name: InsertPublishResult :one
INSERT INTO publish_results (channel_id, idempotency_key, platform_post_id, raw_response)
VALUES ($1, $2, $3, $4)
RETURNING id, channel_id, post_id, idempotency_key, platform_post_id, raw_response, published_at;
