-- Создание таблицы пользователей
CREATE TABLE IF NOT EXISTS users (
                                     id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(100) UNIQUE NOT NULL,
    balance_usd DECIMAL(10,4) NOT NULL DEFAULT 0.0000,
    total_tokens_used BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
                             );

-- Индекс для быстрого поиска по user_id
CREATE INDEX idx_users_user_id ON users(user_id);

-- Вставка тестового пользователя
INSERT INTO users (user_id, balance_usd, total_tokens_used)
VALUES ('test_user_1', 10.0000, 0)
    ON CONFLICT (user_id) DO NOTHING;