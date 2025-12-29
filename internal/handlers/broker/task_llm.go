package broker

import (
	"app/internal/entity/domain"
	apperrors "app/internal/entity/errors"
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/google/uuid"
)

type TaskLLMUCInterface interface {
	UpdateTaskStatus(ctx context.Context, evt domain.TaskLLMStatusEvent) error
}

// TODO вынести в другой namespace
func (h *KafkaMessageHandler) HandleTaskStatusLLM(ctx context.Context, msg *kafka.Message) error {
	h.l.Debug("handling task status LLM message",
		"topic", *msg.TopicPartition.Topic,
		"partition", msg.TopicPartition.Partition,
		"offset", msg.TopicPartition.Offset,
		"key", string(msg.Key),
	)
	// Обрезаем пробельные символы из ключа перед парсингом UUID
	taskIDStr := strings.TrimSpace(string(msg.Key))
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		h.l.Error("failed to parse task_id from message key",
			"error", err,
			"key", string(msg.Key),
			"topic", *msg.TopicPartition.Topic,
		)
		return apperrors.Wrap(apperrors.CodeInvalidMessageFormat, "invalid task_id format", err)
	}

	h.l.Info("processing task status LLM event",
		"topic", *msg.TopicPartition.Topic,
		"task_id", taskID,
		"partition", msg.TopicPartition.Partition,
		"offset", msg.TopicPartition.Offset,
	)

	// 1. Парсим TaskLLMStatusEvent из value
	var kafkaEvent domain.TaskLLMStatusEvent
	if len(msg.Value) > 0 {
		if err := json.Unmarshal(msg.Value, &kafkaEvent.Value); err != nil {
			h.l.Error("failed to unmarshal task status event",
				"error", err,
				"topic", *msg.TopicPartition.Topic,
				"task_id", taskID,
				"value_length", len(msg.Value),
			)
			return apperrors.Wrap(apperrors.CodeInvalidMessageFormat, "failed to unmarshal task status event", err)
		}
	}

	//TODO решить проблему с конвертацией статуса из строки в код Kafka и из kafka в int, нужно сократить путь конвертации до TaskStatus
	// 2. Заполняем из headers (status, worker_id, trace_id) и key (task_id)
	status, workerID, traceID, err := parseTaskStatusHeaders(msg.Headers)
	if err != nil {
		return err
	}

	// Преобразуем TaskStatus в числовой код Kafka
	statusInt := status.ToKafkaCode()

	// Обновляем данные в событии из заголовков и ключа
	kafkaEvent.Key.TaskID = taskID
	kafkaEvent.Headers.Status = statusInt
	if workerID != "" {
		kafkaEvent.Headers.WorkerID = workerID
	}
	if traceID != nil {
		kafkaEvent.Headers.TraceID = traceID
	}

	return h.uc.TaskLLMUC.UpdateTaskStatus(ctx, kafkaEvent)
}

// decodeHeaderValue декодирует значение заголовка из base64 или возвращает как есть
func decodeHeaderValue(value []byte) string {
	decoded, err := base64.StdEncoding.DecodeString(string(value))
	if err == nil {
		return strings.TrimSpace(string(decoded))
	}
	// Если не base64, возвращаем как обычную строку
	return strings.TrimSpace(string(value))
}
