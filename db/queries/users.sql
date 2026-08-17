-- name: GetUserByEmail :one
SELECT id, full_name, email, password
FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT id, full_name, email, password
FROM users
WHERE id = $1;

-- name: CreateUser :one
INSERT INTO users (id, full_name, email, password)
VALUES (gen_random_uuid(), $1, $2, $3)
RETURNING id;

-- name: UpdatePassword :exec
UPDATE users
SET password = $1
WHERE id = $2;

-- name: SaveRefreshToken :exec
INSERT INTO refresh_tokens (jti, user_id, expires_at)
VALUES ($1, $2, $3);

-- name: CheckRefreshToken :one
SELECT EXISTS(
    SELECT 1 FROM refresh_tokens WHERE jti = $1
);

-- name: RevokeRefreshToken :exec
DELETE FROM refresh_tokens
WHERE jti = $1;

-- name: BulkCreateUsers :copyfrom
INSERT INTO users (id, full_name, email, password)
VALUES ($1, $2, $3, $4);
