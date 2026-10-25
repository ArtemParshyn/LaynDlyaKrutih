CREATE TABLE IF NOT EXISTS plan_events (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    date DATE NOT NULL,
    amount BIGINT NOT NULL,
    is_income BOOLEAN NOT NULL,
    category VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    recurring BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_plan_events_user_date ON plan_events(user_id, date);
