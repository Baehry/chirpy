-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (token, created_at, updated_at, user_id, expires_at)
VALUES (
    $1,
    NOW(),
    NOW(),
    $2,
    NOW() + INTERVAL '60 days'
)
RETURNING *;

-- name: GetUserFromRefreshToken :one
SELECT * FROM users
WHERE id = (
    SELECT user_id FROM refresh_tokens
    WHERE token = $1
);

-- name: CheckValidToken :one
SELECT (revoked_at IS NULL AND expires_at > NOW()) AS Valid FROM refresh_tokens;

-- name: RevokeToken :exec
UPDATE refresh_tokens
SET updated_at = NOW(),
revoked_at = NOW()
WHERE token = $1;