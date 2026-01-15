-- name: CreateMatchSession :one
INSERT INTO match_sessions(
    id, 
    created_at, 
    leftbird_id, 
    rightbird_id, 
    session_token, 
    expires_at, 
    user_ip, 
    user_agent
)
VALUES ( 
    gen_random_uuid(),
    NOW(),
    $1,
    $2,
    $3,
    $4,
    $5,
    $6
) RETURNING *;

-- name: GetMatchSessionById :one
SELECT * FROM match_sessions
WHERE id = $1;

-- name: GetMatchSessionByToken :one
SELECT * FROM match_sessions
WHERE session_token = $1;

-- name: VoteMatch :one
UPDATE match_sessions
SET voted = TRUE,
    voted_at = NOW(),
    winnerbird_id = $2
WHERE id = $1
RETURNING *;

-- name: CleanUpSessions :exec
DELETE FROM match_sessions
WHERE expires_at < NOW() - INTERVAL '1 day';