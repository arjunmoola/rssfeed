-- +goose Up
CREATE TABLE feeds (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    name VARCHAR NOT NULL,
    url VARCHAR NOT NULL UNIQUE,
    user_id UUID REFERENCES users ON DELETE CASCADE NOT NULL
);

-- +goose Down
Drop Table feeds;
