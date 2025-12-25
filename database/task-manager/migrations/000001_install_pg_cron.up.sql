-- Создание расширения при первом запуске
CREATE EXTENSION IF NOT EXISTS pg_cron;

-- Предоставление прав текущему пользователю сессии
GRANT USAGE ON SCHEMA cron TO current_user;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA cron TO current_user;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA cron TO current_user;

