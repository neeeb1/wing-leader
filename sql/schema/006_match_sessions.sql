-- +goose Up
CREATE TABLE match_sessions (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP NOT NULL,
    leftbird_id UUID NOT NULL REFERENCES birds,
    rightbird_id UUID NOT NULL REFERENCES birds,
    session_token VARCHAR(64) NOT NULL UNIQUE,
    expires_at TIMESTAMP NOT NULL,
    voted BOOLEAN DEFAULT FALSE,
    voted_at TIMESTAMP,
    winnerbird_id UUID REFERENCES birds,
    user_ip VARCHAR(48),
    user_agent TEXT
);

CREATE INDEX idx_match_session_tokens ON match_sessions(session_token);
CREATE INDEX idx_match_session_expires ON match_sessions(expires_at) WHERE NOT voted;

-- +goose Down
DROP TABLE match_sessions;