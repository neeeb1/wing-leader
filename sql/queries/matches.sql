-- name: RecordMatch :one
INSERT INTO matches (id, created_at, updated_at, winnerbird_id, loserbird_id)
VALUES (
  gen_random_uuid(),
  NOW(),
  NOW(),
  $1,
  $2
) RETURNING *;

-- name: GetMatchByID :one
SELECT * from matches
WHERE id = $1;

-- name: GetAllMatches :many
SELECT * from matches
ORDER BY created_at DESC
LIMIT $1;

-- name: GetMatchByParticipants :many
SELECT * from matches
WHERE (winnerbird_id = $1 OR loserbird_id = $1);