-- ============================================================================
-- Базовая таблица tasks (общая для всех типов задач)
-- ============================================================================
CREATE TABLE tasks (
    -- Идентификация
    task_id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- Состояние
    status        VARCHAR(50) NOT NULL DEFAULT 'PENDING',
    -- PENDING → PROCESSING → COMPLETED | FAILED
    result JSONB, -- Worker info (заполняется при обработке)
    worker_id     VARCHAR(100), -- ID pod'а или worker instance
    
    created_at    TIMESTAMP NOT NULL DEFAULT NOW(), -- Временная метка создания задачи
    started_at    TIMESTAMP, -- Временная метка начала обработки задачи
    completed_at  TIMESTAMP, -- Временная метка завершения задачи
    
    error_message TEXT, -- Сообщение об ошибке
    
    request_id    VARCHAR(100), -- HTTP request ID (для логов gateway)
    trace_id      VARCHAR(100), -- Distributed tracing ID (сквозной)
    
    metadata JSONB, -- Метаданные задачи
    -- Constraints
    CONSTRAINT tasks_status_check 
        CHECK (status IN ('PENDING', 'PROCESSING', 'COMPLETED', 'FAILED', 'CANCELLED')),
    
    CONSTRAINT tasks_date_check 
        CHECK (
            (started_at IS NULL OR started_at >= created_at) AND
            (completed_at IS NULL OR completed_at >= created_at)
        )
);

-- Индексы
CREATE INDEX idx_tasks_status ON tasks(status) WHERE status IN ('PENDING', 'PROCESSING');
CREATE INDEX idx_tasks_created_at ON tasks(created_at DESC);
CREATE INDEX idx_tasks_trace_id ON tasks(trace_id) WHERE trace_id IS NOT NULL;

COMMENT ON TABLE tasks IS 'Базовая таблица задач (общая для всех типов)';
COMMENT ON COLUMN tasks.request_id IS 'ID HTTP запроса клиента (для логов)';
COMMENT ON COLUMN tasks.trace_id IS 'ID операции (OpenTelemetry/Jaeger)';
COMMENT ON COLUMN tasks.worker_id IS 'ID pod''а или worker instance';

COMMENT ON COLUMN tasks.created_at IS 'Временная метка создания задачи';

COMMENT ON COLUMN tasks.started_at IS 'Временная метка начала обработки задачи';

COMMENT ON COLUMN tasks.completed_at IS 'Временная метка завершения задачи';

COMMENT ON COLUMN tasks.error_message IS 'Сообщение об ошибке';

COMMENT ON COLUMN tasks.metadata IS 'Метаданные задачи';

COMMENT ON COLUMN tasks.result IS 'Результат выполнения задачи';



