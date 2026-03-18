package logger

import (
	"context"
)

// WithRequestID добавляет request_id в контекст
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// WithTraceID добавляет trace_id в контекст
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIDKey, traceID)
}

// WithTaskID добавляет task_id в контекст
func WithTaskID(ctx context.Context, taskID string) context.Context {
	return context.WithValue(ctx, TaskIDKey, taskID)
}

// WithWorkerID добавляет worker_id в контекст
func WithWorkerID(ctx context.Context, workerID string) context.Context {
	return context.WithValue(ctx, WorkerIDKey, workerID)
}

// WithCorrelationIDs добавляет все корреляционные ID в контекст
func WithCorrelationIDs(ctx context.Context, requestID, traceID, taskID, workerID string) context.Context {
	ctx = WithRequestID(ctx, requestID)
	ctx = WithTraceID(ctx, traceID)
	ctx = WithTaskID(ctx, taskID)
	ctx = WithWorkerID(ctx, workerID)
	return ctx
}
