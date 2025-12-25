CREATE TABLE IF NOT EXISTS operation_logs (
    id SERIAL PRIMARY KEY,
    task_id VARCHAR(255) NOT NULL,
    trace_id VARCHAR(255) NOT NULL,
    operation VARCHAR(100) NOT NULL,
    metadata JSONB DEFAULT '{}',
    timestamp TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for fast queries
CREATE INDEX IF NOT EXISTS idx_operation_logs_task_id ON operation_logs(task_id);
CREATE INDEX IF NOT EXISTS idx_operation_logs_trace_id ON operation_logs(trace_id);
CREATE INDEX IF NOT EXISTS idx_operation_logs_timestamp ON operation_logs(timestamp);
CREATE INDEX IF NOT EXISTS idx_operation_logs_operation ON operation_logs(operation);