-- +goose Up
ALTER TABLE users add COLUMN role VARCHAR(50) NOT NULL DEFAULT 'user';

-- +goose Down
ALTER TABLE users DROP COLUMN role;
