-- name: PopulateRating :exec
INSERT INTO ratings (
    id,
    created_at,
    updated_at,
    matches,
    rating,
    bird_id
) VALUES (
    gen_random_uuid(),
    NOW(),
    NOW(),
    $1,
    $2,
    $3
) ON CONFLICT (bird_id) DO NOTHING;

-- name: Danger_ResetRatingsDB :exec
UPDATE ratings
set rating = 1000,
updated_at = NOW(),
created_at = NOW(),
matches = 0;

-- name: GetRatingByBirdID :one
SELECT * from ratings
WHERE bird_id = $1;

-- name: UpdateRatingByBirdID :one
UPDATE ratings
set rating = $1,
updated_at = NOW(),
matches = matches + 1
WHERE bird_id = $2
RETURNING *;

-- name: GetTopRatings :many
SELECT ratings.id, ratings.matches, ratings.rating, birds.common_name, birds.scientific_name, birds.status from ratings
INNER JOIN birds ON ratings.bird_id = birds.id
ORDER BY rating DESC
LIMIT $1;

-- name: GetTotalRatings :one
SELECT count(*) from ratings;