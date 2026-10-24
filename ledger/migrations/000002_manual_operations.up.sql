CREATE TABLE IF NOT EXISTS manual_operations (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    date DATE NOT NULL,
    amount BIGINT NOT NULL,
    category VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_manual_operations_user_date ON manual_operations(user_id, date);
