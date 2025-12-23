-- ============================================================================
-- Базовая таблица tasks (общая для всех типов задач)
-- ============================================================================
CREATE TABLE tasks (
    -- Идентификация
    task_id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- Состояние
    status        VARCHAR(50) NOT NULL DEFAULT 'PENDING',
    -- PENDING → PROCESSING → COMPLETED | FAILED
    result JSONB,
    -- Worker info (заполняется при обработке)
    worker_id     VARCHAR(100),  -- ID pod'а или worker instance
    -- Временные метки (для метрик и SLA)
    created_at    TIMESTAMP NOT NULL DEFAULT NOW(),
    started_at    TIMESTAMP,      -- когда worker начал обработку
    completed_at  TIMESTAMP,      -- когда завершилась (успех или ошибка)
    
    -- Ошибки
    error_message TEXT,
    
    -- Трейсинг (observability)
    request_id    VARCHAR(100),   -- HTTP request ID (для логов gateway)
    trace_id      VARCHAR(100),   -- Distributed tracing ID (сквозной)
    metadata JSONB,
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
COMMENT ON COLUMN tasks.trace_id IS 'Distributed tracing ID (OpenTelemetry/Jaeger)';


