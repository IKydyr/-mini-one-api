-- Таблица для API токенов (аутентификация)
CREATE TABLE IF NOT EXISTS api_tokens (
                                          id SERIAL PRIMARY KEY,
                                          token VARCHAR(255) UNIQUE NOT NULL,
    user_id VARCHAR(100) NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE
                                                                );

-- Индекс для быстрого поиска по токену
CREATE INDEX idx_api_tokens_token ON api_tokens(token);

-- Вставка тестового токена (sk-test123)
INSERT INTO api_tokens (token, user_id, name, expires_at)
VALUES ('sk-test123', 'test_user_1', 'Test Token', NULL);