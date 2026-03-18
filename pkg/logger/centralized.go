package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// CentralizedLoggerConfig конфигурация централизованного логирования
type CentralizedLoggerConfig struct {
	Enabled     bool
	URL         string
	Token       string
	ServiceName string
	Timeout     time.Duration
}

// CentralizedHandler реализует slog.Handler для отправки логов в централизованную систему
type CentralizedHandler struct {
	config CentralizedLoggerConfig
	next   slog.Handler
	client *http.Client
}

// NewCentralizedHandler создает новый обработчик для централизованного логирования
func NewCentralizedHandler(config CentralizedLoggerConfig, next slog.Handler) slog.Handler {
	client := &http.Client{
		Timeout: config.Timeout,
	}

	return &CentralizedHandler{
		config: config,
		next:   next,
		client: client,
	}
}

// Enabled проверяет, включен ли уровень логирования
func (h *CentralizedHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

// Handle обрабатывает запись лога
func (h *CentralizedHandler) Handle(ctx context.Context, record slog.Record) error {
	// Сначала обрабатываем локально
	if err := h.next.Handle(ctx, record); err != nil {
		return err
	}

	// Если централизованное логирование отключено, выходим
	if !h.config.Enabled {
		return nil
	}

	// Отправляем в центральную систему в отдельной горутине, чтобы не блокировать основной поток
	go h.sendToCentralized(record)

	return nil
}

// WithAttrs возвращает новый обработчик с дополнительными атрибутами
func (h *CentralizedHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &CentralizedHandler{
		config: h.config,
		next:   h.next.WithAttrs(attrs),
		client: h.client,
	}
}

// WithGroup возвращает новый обработчик с группой атрибутов
func (h *CentralizedHandler) WithGroup(name string) slog.Handler {
	return &CentralizedHandler{
		config: h.config,
		next:   h.next.WithGroup(name),
		client: h.client,
	}
}

// sendToCentralized отправляет запись в централизованную систему
func (h *CentralizedHandler) sendToCentralized(record slog.Record) {
	logEntry := h.formatLogEntry(record)

	jsonData, err := json.Marshal(logEntry)
	if err != nil {
		// Не можем отправить лог об ошибке отправки, так как это может привести к рекурсии
		return
	}

	req, err := http.NewRequest("POST", h.config.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return
	}

	req.Header.Set("Content-Type", "application/json")
	if h.config.Token != "" {
		req.Header.Set("Authorization", "Bearer "+h.config.Token)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	// Можно добавить логику повторных попыток при необходимости
}

// formatLogEntry форматирует запись лога для отправки в централизованную систему
func (h *CentralizedHandler) formatLogEntry(record slog.Record) map[string]interface{} {
	entry := map[string]interface{}{
		"timestamp":  record.Time.Format(time.RFC3339Nano),
		"level":      record.Level.String(),
		"message":    record.Message,
		"service":    h.config.ServiceName,
		"attributes": make(map[string]interface{}),
	}

	// Добавляем атрибуты
	record.Attrs(func(attr slog.Attr) bool {
		entry["attributes"].(map[string]interface{})[attr.Key] = attr.Value.Any()
		return true
	})

	return entry
}

// LogExporter предоставляет интерфейс для экспорта логов в различные системы
type LogExporter interface {
	Export(logs []map[string]interface{}) error
}

// HTTPLogExporter реализует экспорт логов через HTTP
type HTTPLogExporter struct {
	config CentralizedLoggerConfig
	client *http.Client
}

// NewHTTPLogExporter создает новый HTTP экспортер логов
func NewHTTPLogExporter(config CentralizedLoggerConfig) *HTTPLogExporter {
	return &HTTPLogExporter{
		config: config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// Export экспортирует логи в удаленную систему
func (e *HTTPLogExporter) Export(logs []map[string]interface{}) error {
	if !e.config.Enabled {
		return nil
	}

	jsonData, err := json.Marshal(logs)
	if err != nil {
		return fmt.Errorf("failed to marshal logs: %w", err)
	}

	req, err := http.NewRequest("POST", e.config.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if e.config.Token != "" {
		req.Header.Set("Authorization", "Bearer "+e.config.Token)
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send logs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("export failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// LokiLogExporter реализует экспорт логов в Grafana Loki
type LokiLogExporter struct {
	config CentralizedLoggerConfig
	client *http.Client
}

// NewLokiLogExporter создает новый Loki экспортер логов
func NewLokiLogExporter(config CentralizedLoggerConfig) *LokiLogExporter {
	return &LokiLogExporter{
		config: config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// Export экспортирует логи в Loki
func (e *LokiLogExporter) Export(logs []map[string]interface{}) error {
	if !e.config.Enabled {
		return nil
	}

	// Формируем структуру данных в формате Loki
	streams := make([]map[string]interface{}, 0)

	for _, log := range logs {
		labels := map[string]string{
			"service": e.config.ServiceName,
			"job":     "task-manager",
		}

		// Добавляем дополнительные метки из атрибутов лога
		if attrs, ok := log["attributes"].(map[string]interface{}); ok {
			if category, exists := attrs["category"]; exists {
				labels["category"] = fmt.Sprintf("%v", category)
			}
			if level, exists := attrs["level"]; exists {
				labels["level"] = fmt.Sprintf("%v", level)
			}
		}

		stream := map[string]interface{}{
			"stream": labels,
			"values": [][]string{
				{
					fmt.Sprintf("%d", time.Now().UnixNano()), // timestamp in nanoseconds
					fmt.Sprintf("%v", log["message"]),        // log message
				},
			},
		}
		streams = append(streams, stream)
	}

	data := map[string]interface{}{
		"streams": streams,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal logs for Loki: %w", err)
	}

	req, err := http.NewRequest("POST", e.config.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create Loki request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if e.config.Token != "" {
		req.Header.Set("Authorization", "Bearer "+e.config.Token)
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send logs to Loki: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Loki export failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
