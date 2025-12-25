package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// TODO рассмотреть вараинт использования функционала оркестратора вместо этого файла
// RotatingWriter реализует io.Writer с ротацией логов
type RotatingWriter struct {
	filename     string
	maxSize      int64 // максимальный размер файла в байтах
	maxFiles     int   // максимальное количество файлов
	clearOnStart bool  // очищать ли файл при старте
	currentSize  int64
	file         *os.File
	mu           sync.Mutex
}

// NewRotatingWriter создает новый RotatingWriter
func NewRotatingWriter(filename string, maxSize int64, maxFiles int, clearOnStart bool) (*RotatingWriter, error) {
	rw := &RotatingWriter{
		filename:     filename,
		maxSize:      maxSize,
		maxFiles:     maxFiles,
		clearOnStart: clearOnStart,
	}

	// Создаем директорию, если её нет
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Очищаем файл при старте, если нужно
	if clearOnStart {
		if err := os.Truncate(filename, 0); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to clear log file: %w", err)
		}
	}

	// Открываем файл
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	// Получаем текущий размер файла
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to stat log file: %w", err)
	}

	rw.file = file
	rw.currentSize = info.Size()

	return rw, nil
}

// Write реализует io.Writer
func (rw *RotatingWriter) Write(p []byte) (n int, err error) {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	// Проверяем, нужно ли ротировать
	if rw.currentSize+int64(len(p)) > rw.maxSize && rw.maxSize > 0 {
		if err := rw.rotate(); err != nil {
			return 0, err
		}
	}

	// Записываем данные
	n, err = rw.file.Write(p)
	if err != nil {
		return n, err
	}

	rw.currentSize += int64(n)
	return n, nil
}

// rotate выполняет ротацию файла
func (rw *RotatingWriter) rotate() error {
	// Закрываем текущий файл
	if err := rw.file.Close(); err != nil {
		return fmt.Errorf("failed to close log file: %w", err)
	}

	// Если maxFiles > 0, ротируем существующие файлы
	if rw.maxFiles > 0 {
		// Ротируем существующие файлы
		for i := rw.maxFiles - 1; i > 0; i-- {
			oldName := rw.getRotatedFilename(i)
			newName := rw.getRotatedFilename(i + 1)

			// Удаляем самый старый файл, если он существует
			if i == rw.maxFiles-1 {
				if err := os.Remove(newName); err != nil && !os.IsNotExist(err) {
					return fmt.Errorf("failed to remove old log file: %w", err)
				}
			}

			// Переименовываем файл
			if err := os.Rename(oldName, newName); err != nil && !os.IsNotExist(err) {
				// Игнорируем ошибку, если файл не существует
				continue
			}
		}
	}

	// Переименовываем текущий файл в .1
	rotatedName := rw.getRotatedFilename(1)
	if err := os.Rename(rw.filename, rotatedName); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to rotate log file: %w", err)
	}

	// Создаем новый файл
	file, err := os.OpenFile(rw.filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create new log file: %w", err)
	}

	rw.file = file
	rw.currentSize = 0

	return nil
}

// getRotatedFilename возвращает имя ротированного файла
func (rw *RotatingWriter) getRotatedFilename(index int) string {
	if index == 0 {
		return rw.filename
	}
	return fmt.Sprintf("%s.%d", rw.filename, index)
}

// Close закрывает файл
func (rw *RotatingWriter) Close() error {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	if rw.file != nil {
		return rw.file.Close()
	}
	return nil
}

// Sync синхронизирует файл на диск
func (rw *RotatingWriter) Sync() error {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	if rw.file != nil {
		return rw.file.Sync()
	}
	return nil
}

// RotateNow принудительно выполняет ротацию
func (rw *RotatingWriter) RotateNow() error {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	return rw.rotate()
}
