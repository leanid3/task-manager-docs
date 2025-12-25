-- Удаление cron задачи
SELECT cron.unschedule('check-stuck-tasks');

-- Удаление функции
DROP FUNCTION IF EXISTS check_stuck_tasks_simple();
