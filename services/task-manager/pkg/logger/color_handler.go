package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

type ColorTextHandler struct {
	w           slog.Handler
	levelColors map[string]string
	catColors   map[string]string
	isTerminal  bool
}

func NewColorTextHandler(w io.Writer, opts *slog.HandlerOptions) slog.Handler {
	h := &ColorTextHandler{
		w:           slog.NewTextHandler(w, opts),
		levelColors: ColorLevelMap,
		catColors:   ColorCategoryMap,
		isTerminal:  isTerminal(w),
	}

	if !h.isTerminal {
		// Если не терминал, используем обычный handler
		return slog.NewTextHandler(w, opts)
	}

	return h
}

func (h *ColorTextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.w.Enabled(ctx, level)
}

func (h *ColorTextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *ColorTextHandler) WithGroup(name string) slog.Handler {
	return h
}

func (h *ColorTextHandler) Handle(ctx context.Context, r slog.Record) error {
	// Создаем цветной вывод
	if h.isTerminal {
		coloredLine := h.colorizeRecord(r)
		// Создаем новую запись с цветным сообщением, используя PC из оригинальной записи
		newRecord := slog.NewRecord(r.Time, r.Level, coloredLine, r.PC)
		// Копируем атрибуты из оригинальной записи
		r.Attrs(func(a slog.Attr) bool {
			newRecord.AddAttrs(a)
			return true
		})
		return h.w.Handle(ctx, newRecord)
	}

	return h.w.Handle(ctx, r)
}

func (h *ColorTextHandler) colorizeRecord(r slog.Record) string {
	var builder strings.Builder

	// Время [HH:MM:SS]
	builder.WriteString(fmt.Sprintf("[%s] ", r.Time.Format("15:04:05")))

	// Уровень с цветом
	level := strings.ToUpper(r.Level.String())
	color := h.levelColors[level]
	if color == "" {
		color = ColorGray
	}
	builder.WriteString(fmt.Sprintf("%s%-5s%s ", color, level, ColorReset))

	// Category с цветом
	var category string
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == "category" {
			category = a.Value.String()
			return false // останавливаем итерацию
		}
		return true
	})
	if category != "" {
		catColor := h.catColors[category]
		if catColor == "" {
			catColor = ColorWhite
		}
		builder.WriteString(fmt.Sprintf("%s[%s]%s ", catColor, category, ColorReset))
	}

	// Сообщение
	msg := r.Message
	builder.WriteString(msg)

	// Атрибуты (упрощенный вывод)
	attrs := make([]string, 0)
	r.Attrs(func(a slog.Attr) bool {
		if a.Key != "category" && a.Key != "source" {
			attrs = append(attrs, fmt.Sprintf("%s=%v", a.Key, a.Value))
		}
		return true
	})

	if len(attrs) > 0 {
		builder.WriteString(" | ")
		builder.WriteString(strings.Join(attrs, " "))
	}

	return builder.String()
}

func isTerminal(w io.Writer) bool {
	// Проверяем, является ли writer терминалом
	switch v := w.(type) {
	case *os.File:
		return termIsatty(int(v.Fd()))
	}
	return false
}

// termIsatty имитирует проверку терминала (можно использовать golang.org/x/term)
func termIsatty(fd int) bool {
	// Простая проверка для fd 1 (stdout) и 2 (stderr)
	return fd == 1 || fd == 2
}
