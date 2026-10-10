-- おうちと家族（docs/domain/household.md）

-- +goose Up
CREATE TABLE households (
    id         uuid        PRIMARY KEY,
    created_at timestamptz NOT NULL
);

-- 家族の役割とようすは、ドメインのコード（RoleKeeper など）に対応する文字列で持つ。
CREATE TABLE members (
    id uuid PRIMARY KEY,
    household_id uuid NOT NULL REFERENCES households (id) ON DELETE RESTRICT,
    role text NOT NULL CHECK (role IN ('keeper', 'non_keeper')),
    status text NOT NULL CHECK (status IN ('active', 'left')),
    created_at timestamptz NOT NULL
);

CREATE INDEX members_household_id_idx ON members (household_id);

-- +goose Down
DROP TABLE members;
DROP TABLE households;