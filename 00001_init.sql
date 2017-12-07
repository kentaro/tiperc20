-- +goose Up
CREATE TABLE accounts (
    id SERIAL,
    slack_user_id TEXT UNIQUE,
    ethereum_address TEXT
);

-- +goose Down
DROP TABLE accounts;
