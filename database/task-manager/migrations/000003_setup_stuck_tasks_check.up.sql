-- Создание функции
CREATE OR REPLACE FUNCTION check_stuck_tasks_simple()
RETURNS INTEGER AS $$
DECLARE
    updated_count INTEGER;
BEGIN
    UPDATE tasks
    SET 
        status = 'CANCELLED',
        completed_at = NOW(),
        error_message = 'Не удалось выполнить задачу, повторите еще раз'
    WHERE 
        (status = 'PENDING' AND created_at < NOW() - INTERVAL '1 hours')
        OR
        (status = 'PROCESSING' AND started_at IS NOT NULL AND started_at < NOW() - INTERVAL '1 hours');

    GET DIAGNOSTICS updated_count = ROW_COUNT;
    RETURN updated_count;
END;
$$ LANGUAGE plpgsql;

-- Удаление существующей задачи (для idempotency)
-- Используем DO блок для безопасного удаления, если задача существует
DO $$
BEGIN
    -- Пытаемся удалить задачу, если она существует
    -- cron.unschedule возвращает количество удаленных задач (0, если не найдено)
    PERFORM cron.unschedule('check-stuck-tasks');
EXCEPTION
    WHEN undefined_function THEN
        -- Игнорируем ошибку, если функция не существует (расширение не установлено)
        NULL;
    WHEN OTHERS THEN
        -- Игнорируем другие ошибки (например, задача не существует)
        NULL;
END $$;

-- Создание cron задачи
-- Используем значение по умолчанию 5 минут, если переменная окружения не установлена
-- Формат cron: '*/5 * * * *' означает каждые 5 минут
SELECT cron.schedule(
    'check-stuck-tasks',
    '*/5 * * * *',
    'SELECT check_stuck_tasks_simple();'
);
