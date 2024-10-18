DROP TABLE IF EXISTS referral_links CASCADE; -- Удаляем реферальные ссылки
DROP TABLE IF EXISTS referral_codes CASCADE; -- Удаляем реферальные коды
DROP TABLE IF EXISTS users CASCADE;           -- Удаляем пользователей

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL,
    email VARCHAR(100) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,   -- Хэш пароля
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);


CREATE TABLE referral_codes (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code VARCHAR(50) NOT NULL UNIQUE, -- Уникальный реферальный код
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL, -- Срок действия кода
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE referral_links (
    id SERIAL PRIMARY KEY,
    referrer_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE, -- Кто пригласил
    referee_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE, -- Кто зарегистрировался
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Индекс для ускорения поиска рефералов по рефереру
CREATE INDEX idx_referral_links_referrer_id ON referral_links(referrer_id);