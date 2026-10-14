-- name: CreateHousehold :exec
INSERT INTO households (id, created_at)
VALUES ($1, $2);

-- name: LockHousehold :one
-- おうちの行をロックし、同じおうちの操作を 1 つずつ順に扱う（ADR-0007）。
SELECT id, created_at
FROM households
WHERE id = $1
FOR UPDATE;

-- name: CreateMember :exec
INSERT INTO members (id, household_id, role, status, created_at)
VALUES ($1, $2, $3, $4, $5);

-- name: ListMembers :many
SELECT id, household_id, role, status, created_at
FROM members
WHERE household_id = $1
ORDER BY created_at, id;

-- name: UpdateMember :exec
UPDATE members
SET role = $2, status = $3
WHERE id = $1;
