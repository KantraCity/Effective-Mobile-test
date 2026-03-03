CREATE TABLE IF NOT EXISTS subscriptions (
    id           SERIAL PRIMARY KEY,
    service_name VARCHAR(255)             NOT NULL,
    price        INTEGER                  NOT NULL CHECK (price > 0),
    user_id      UUID                     NOT NULL,
    start_date   DATE                     NOT NULL,
    end_date     DATE,
    created_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_subs_user_service ON subscriptions (user_id, service_name);