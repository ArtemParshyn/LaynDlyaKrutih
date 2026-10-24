CREATE TABLE IF NOT EXISTS operations (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    date DATE NOT NULL,
    amount BIGINT NOT NULL,
    category VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    recurring BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE INDEX IF NOT EXISTS idx_operations_user_date ON operations(user_id, date);

CREATE TABLE IF NOT EXISTS statements (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    filename VARCHAR(255) NOT NULL,
    uploaded_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_statements_user_id ON statements(user_id);

CREATE TABLE IF NOT EXISTS forecast_days (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    date DATE NOT NULL,
    expected_balance BIGINT NOT NULL,
    balance_low BIGINT NOT NULL,
    balance_high BIGINT NOT NULL,
    inflow BIGINT NOT NULL,
    outflow BIGINT NOT NULL,
    rough BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE INDEX IF NOT EXISTS idx_forecast_days_user_id ON forecast_days(user_id);

CREATE TABLE IF NOT EXISTS forecast_alerts (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    date DATE NOT NULL,
    shortfall BIGINT NOT NULL,
    chance DOUBLE PRECISION NOT NULL,
    reason TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_forecast_alerts_user_id ON forecast_alerts(user_id);

CREATE TABLE IF NOT EXISTS forecast_history (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    date DATE NOT NULL,
    expected_balance BIGINT NOT NULL,
    balance_low BIGINT NOT NULL,
    balance_high BIGINT NOT NULL,
    inflow BIGINT NOT NULL,
    outflow BIGINT NOT NULL,
    rough BOOLEAN NOT NULL DEFAULT FALSE,
    archived_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_forecast_history_user_id ON forecast_history(user_id);
