package config

import (
	"app/pkg/logger"
	"context"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// Manager управляет конфигурацией приложения с поддержкой горячей перезагрузки
type Manager struct {
	configMu sync.RWMutex
	config   *Config
	logger   logger.Interface
	watcher  *fsnotify.Watcher
	cancel   context.CancelFunc
}

// NewManager создает новый менеджер конфигурации
func NewManager(l logger.Interface) (*Manager, error) {
	cfg, err := Load(l)
	if err != nil {
		return nil, err
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	manager := &Manager{
		config:  cfg,
		logger:  l,
		watcher: watcher,
	}

	return manager, nil
}

// LoadConfig загружает конфигурацию
func (m *Manager) LoadConfig() (*Config, error) {
	newConfig, err := Load(m.logger)
	if err != nil {
		return nil, err
	}

	m.configMu.Lock()
	m.config = newConfig
	m.configMu.Unlock()

	return newConfig, nil
}

// Get возвращает текущую конфигурацию
func (m *Manager) Get() *Config {
	m.configMu.RLock()
	defer m.configMu.RUnlock()
	return m.config
}

// Watch запускает наблюдение за изменениями в файле конфигурации
func (m *Manager) Watch(ctx context.Context, configFile string) error {
	if err := m.watcher.Add(configFile); err != nil {
		return err
	}

	// Создаем контекст для отмены горутины
	ctx, cancel := context.WithCancel(ctx)
	m.cancel = cancel

	go func() {
		defer m.watcher.Close()

		for {
			select {
			case <-ctx.Done():
				m.logger.Info("configuration watcher stopped")
				return
			case event, ok := <-m.watcher.Events:
				if !ok {
					return
				}

				if event.Op&fsnotify.Write == fsnotify.Write {
					m.logger.Info("configuration file changed, reloading...", "file", event.Name)

					if _, err := m.LoadConfig(); err != nil {
						m.logger.Error("failed to reload configuration", "error", err)
						continue
					}

					m.logger.Info("configuration reloaded successfully")
				}
			case err, ok := <-m.watcher.Errors:
				if !ok {
					return
				}
				m.logger.Error("configuration watcher error", "error", err)
			}
		}
	}()

	return nil
}

// Stop останавливает наблюдение за файлом конфигурации
func (m *Manager) Stop() {
	if m.cancel != nil {
		m.cancel()
	}
}

// GetTaskConfig возвращает конфигурацию для конкретного типа задачи
func (m *Manager) GetTaskConfig(taskType string) *TaskConfig {
	m.configMu.RLock()
	defer m.configMu.RUnlock()

	for _, taskConfig := range m.config.Tasks {
		if taskConfig.Type == taskType && taskConfig.Enabled {
			return &taskConfig
		}
	}

	return nil
}

// GetAllTaskConfigs возвращает все активные конфигурации задач
func (m *Manager) GetAllTaskConfigs() []TaskConfig {
	m.configMu.RLock()
	defer m.configMu.RUnlock()

	var activeConfigs []TaskConfig
	for _, taskConfig := range m.config.Tasks {
		if taskConfig.Enabled {
			activeConfigs = append(activeConfigs, taskConfig)
		}
	}

	return activeConfigs
}
