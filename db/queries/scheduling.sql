-- name: CreateScheduledJob :one
INSERT INTO scheduled_jobs (post_id, channel_id, run_at, status)
VALUES ($1, $2, $3, $4)
RETURNING id, post_id, channel_id, run_at, status, attempts, last_error, asynq_task_id, created_at, updated_at;

-- name: GetScheduledJob :one
SELECT id, post_id, channel_id, run_at, status, attempts, last_error, asynq_task_id, created_at, updated_at
FROM scheduled_jobs
WHERE id = $1;

-- name: SetScheduledJobTaskID :exec
UPDATE scheduled_jobs
SET asynq_task_id = $2, updated_at = now()
WHERE id = $1;

-- name: SetScheduledJobStatus :exec
UPDATE scheduled_jobs
SET status = $2, last_error = $3, attempts = attempts + $4, updated_at = now()
WHERE id = $1;

-- ClaimScheduledJob atomically transitions scheduled -> publishing (counting the
-- attempt), returning the row only if the claim succeeded. A canceled/published
-- job is NOT scheduled, so it cannot be claimed — the worker then skips it.
-- name: ClaimScheduledJob :one
UPDATE scheduled_jobs
SET status = 'publishing', attempts = attempts + 1, updated_at = now()
WHERE id = $1 AND status = 'scheduled'
RETURNING id;

-- name: CancelScheduledJob :execrows
UPDATE scheduled_jobs
SET status = 'canceled', updated_at = now()
WHERE id = $1 AND status = 'scheduled';

-- name: ListScheduledJobsInRange :many
SELECT j.id, j.post_id, j.channel_id, j.run_at, j.status, j.attempts, j.last_error, j.asynq_task_id, j.created_at, j.updated_at
FROM scheduled_jobs j
JOIN channels c ON c.id = j.channel_id
WHERE c.workspace_id = $1 AND j.run_at >= $2 AND j.run_at < $3
ORDER BY j.run_at;

-- name: ListSlotsForChannel :many
SELECT id, channel_id, day_of_week, time_of_day, timezone, created_at
FROM schedule_slots
WHERE channel_id = $1
ORDER BY day_of_week, time_of_day;

-- name: CreateScheduleSlot :one
INSERT INTO schedule_slots (channel_id, day_of_week, time_of_day, timezone)
VALUES ($1, $2, $3, $4)
RETURNING id, channel_id, day_of_week, time_of_day, timezone, created_at;

-- name: DeleteScheduleSlot :exec
DELETE FROM schedule_slots
WHERE id = $1;

-- name: ListScheduledRunAtForChannel :many
SELECT run_at
FROM scheduled_jobs
WHERE channel_id = $1 AND status = 'scheduled' AND run_at >= $2
ORDER BY run_at;

-- name: CountPendingJobsForWorkspace :one
-- Jobs not yet in a terminal state, for the per-workspace queue quota.
SELECT count(*) FROM scheduled_jobs sj
JOIN posts p ON p.id = sj.post_id
WHERE p.workspace_id = $1 AND sj.status IN ('scheduled', 'publishing');
