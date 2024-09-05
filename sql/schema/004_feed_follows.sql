-- +goose Up
CREATE TABLE feed_follows (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    user_id UUID REFERENCES users ON DELETE CASCADE NOT NULL,
    feed_id UUID REFERENCES feeds ON DELETE CASCADE NOT NULL
);

-- +goose Down
DROP TABLE feed_follows;
