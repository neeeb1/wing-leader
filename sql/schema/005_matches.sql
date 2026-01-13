-- +goose Up
CREATE TABLE matches (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    winnerbird_id UUID NOT NULL REFERENCES birds
    ON DELETE CASCADE,
    loserbird_id UUID NOT NULL REFERENCES birds
    ON DELETE CASCADE,
    FOREIGN KEY (winnerbird_id)
    REFERENCES birds(id),
    FOREIGN KEY (loserbird_id)
    REFERENCES birds(id)
);

-- +goose Down
DROP TABLE matches;