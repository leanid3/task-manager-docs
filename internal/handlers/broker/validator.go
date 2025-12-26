package broker

import (
	"app/internal/entity/domain"
	apperrors "app/internal/entity/errors"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/google/uuid"
)

func validateTaskID(key []byte) (uuid.UUID, error) {
	taskIDStr := strings.TrimSpace(string(key))
	if taskIDStr == "" {
		return uuid.Nil, apperrors.New(apperrors.CodeInvalidMessageFormat, "invalid task_id format")
	}
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		return uuid.Nil, apperrors.Wrap(apperrors.CodeInvalidMessageFormat, "invalid task_id format", err)
	}
	return taskID, nil
}

func validateStatus(headers []kafka.Header) (int, error) {
	var statusStr string
	for _, header := range headers {
		if string(header.Key) == "status" {
			statusStr = decodeHeaderValue(header.Value)
			break
		}
	}
	if statusStr == "" {
		return 0, apperrors.New(apperrors.CodeInvalidMessageFormat, "invalid task status format")
	}
	statusInt, err := strconv.Atoi(statusStr)
	if err != nil {
		return 0, apperrors.Wrap(apperrors.CodeInvalidMessageFormat, "invalid task status format", err)
	}
	_, ok := domain.TaskStatus("").FromKafkaCode(statusInt)
	if !ok {
		return 0, apperrors.New(apperrors.CodeInvalidMessageFormat, "invalid task status")
	}
	return statusInt, nil
}

// TODO нужен рефакторинг
func parseTaskStatusHeaders(headers []kafka.Header) (domain.TaskStatus, string, *uuid.UUID, error) {
	var status domain.TaskStatus
	var workerID string
	var traceID *uuid.UUID

	for _, header := range headers {
		switch string(header.Key) {
		case "status":
			statusStr := decodeHeaderValue(header.Value)
			if statusStr == "" {
				return "", "", nil, apperrors.New(apperrors.CodeInvalidMessageFormat, "invalid task status format")
			}
			statusInt, err := strconv.Atoi(statusStr)
			if err != nil {
				return "", "", nil, apperrors.Wrap(apperrors.CodeInvalidMessageFormat, "invalid task status format", err)
			}
			var ok bool
			status, ok = domain.TaskStatus("").FromKafkaCode(statusInt)
			if !ok {
				return "", "", nil, apperrors.New(apperrors.CodeInvalidMessageFormat, "invalid task status")
			}
		case "worker_id":
			workerID = decodeHeaderValue(header.Value)
		case "trace_id":
			traceIDStr := decodeHeaderValue(header.Value)
			if traceIDStr != "" {
				id, err := uuid.Parse(traceIDStr)
				if err != nil {
					return "", "", nil, apperrors.Wrap(apperrors.CodeInvalidMessageFormat, "invalid trace_id format", err)
				}
				traceID = &id
			}
		}
	}

	return status, workerID, traceID, nil
}

func ValidateTaskStatusLLM(msg *kafka.Message) (*domain.TaskLLMStatusEvent, error) {
	// Валидация task_id
	taskID, err := validateTaskID(msg.Key)
	if err != nil {
		return nil, err
	}

	// Валидация статуса
	statusInt, err := validateStatus(msg.Headers)
	if err != nil {
		return nil, err
	}

	// Валидация JSON
	var value domain.TaskLLMStatusEventPayload
	if len(msg.Value) > 0 {
		if err := json.Unmarshal(msg.Value, &value); err != nil {
			return nil, apperrors.Wrap(apperrors.CodeInvalidMessageFormat, "failed to unmarshal task status event", err)
		}
	}

	// Парсинг заголовков
	_, workerID, traceID, err := parseTaskStatusHeaders(msg.Headers)
	if err != nil {
		return nil, err
	}

	// Создание события
	event := &domain.TaskLLMStatusEvent{
		Key: domain.TaskContractKey{TaskID: taskID},
		Headers: domain.TaskContractHeaders{
			Status:   statusInt,
			WorkerID: workerID,
			TraceID:  traceID,
		},
		Value: value,
	}

	return event, nil
}
