-- Создание расширения при первом запуске
CREATE EXTENSION IF NOT EXISTS pg_cron;

-- Предоставление прав обычным пользователям 
GRANT USAGE ON SCHEMA cron TO postgres;