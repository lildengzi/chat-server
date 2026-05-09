CREATE TABLE users (
    user_id BIGSERIAL PRIMARY KEY,
    username VARCHAR(64) UNIQUE NOT NULL,
    password TEXT NOT NULL
);

CREATE TABLE friends (
    user_id BIGINT NOT NULL,
    friend_id BIGINT NOT NULL,
    UNIQUE (user_id, friend_id)
);

CREATE TABLE offline_messages (
    msg_id BIGSERIAL PRIMARY KEY,
    to_user_id BIGINT NOT NULL,
    from_user_id BIGINT NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_offline_messages_to_user_id ON offline_messages(to_user_id);
