.PHONY: help help-vars h hv tu ti ta bd rd pb pr rm rc ra
.PHONY: test-unit test-integration test-all build-dev run-dev
.PHONY: prod-build prod-run rm-image rm-container rm-all
.PHONY: logs status stop restart run-container
.PHONY: deps deps-tidy deps-download deps-verify deps-update
.PHONY: fmt vet lint swagger swagger-clean
.PHONY: build-local run-local clean clean-all clean-logs
.PHONY: check pre-commit shell exec version info config-check

# =============================================================================
# Variables Configuration
# =============================================================================

# Default values (можно переопределить через CLI или окружение)
NAME_C ?= task-manager-app
PORT_C ?= 8080
MPORT_C ?= 9090
NET_C ?= bridge
NOCACHE_C ?= false
CACHEFROM_C ?= task-manager:dev
LOGS_C ?= ./logs
BINARY_NAME ?= task-manager
SWAGGER_DIR ?= docs

# Internal variables (используются в командах)
container_name := $(NAME_C)
container_port := $(PORT_C)
container_metrics_port := $(MPORT_C)
container_network := $(NET_C)
container_build_no_cache := $(NOCACHE_C)
container_build_cache_from := $(CACHEFROM_C)
logs_volume := $(LOGS_C)

# Build arguments
container_build_args := --build-arg HTTP_PORT=$(container_port) \
                        --build-arg METRICS_PORT=$(container_metrics_port) \
                        --build-arg CONFIG_FILE=config.yaml

# Cache logic
CACHE_OPT = $(if $(filter true 1 yes,$(container_build_no_cache)),--no-cache,--cache-from=$(container_build_cache_from))

# Colors for help (определяем один раз)
BOLD := \033[1m
GREEN := \033[1;32m
YELLOW := \033[1;33m
BLUE := \033[1;34m
CYAN := \033[1;36m
RESET := \033[0m

# =============================================================================
# Help Commands
# =============================================================================

help:
	@printf "$(CYAN)%-20s$(RESET) %s\n" "Makefile commands:" ""
	@echo ""
	@printf "$(BOLD)$(CYAN)Development:$(RESET)\n"
	@printf "$(GREEN)%-20s$(RESET) %s\n" "make build-dev" "- собрать образ в dev среде"
	@printf "$(GREEN)%-20s$(RESET) %s\n" "make run-dev" "- запустить контейнер в dev среде"
	@printf "$(GREEN)%-20s$(RESET) %s\n" "make build-local" "- собрать локальный бинарник"
	@printf "$(GREEN)%-20s$(RESET) %s\n" "make run-local" "- запустить локально (без Docker)"
	@echo ""
	@printf "$(BOLD)$(CYAN)Production:$(RESET)\n"
	@printf "$(GREEN)%-20s$(RESET) %s\n" "make prod-build" "- собрать образ в prod среде"
	@printf "$(GREEN)%-20s$(RESET) %s\n" "make prod-run" "- запустить контейнер в prod среде"
	@echo ""
	@printf "$(BOLD)$(CYAN)Testing:$(RESET)\n"
	@printf "$(GREEN)%-20s$(RESET) %s\n" "make test-unit" "- запустить unit тесты"
	@printf "$(GREEN)%-20s$(RESET) %s\n" "make test-integration" "- запустить integration тесты"
	@printf "$(GREEN)%-20s$(RESET) %s\n" "make test-all" "- запустить все тесты"
	@echo ""
	@printf "$(BOLD)$(CYAN)Code Quality:$(RESET)\n"
	@printf "$(GREEN)%-20s$(RESET) %s\n" "make fmt" "- форматировать код"
	@printf "$(GREEN)%-20s$(RESET) %s\n" "make vet" "- проверить код (go vet)"
	@printf "$(GREEN)%-20s$(RESET) %s\n" "make check" "- проверить код перед коммитом"
	@printf "$(GREEN)%-20s$(RESET) %s\n" "make swagger" "- сгенерировать swagger документацию"
	@echo ""
	@printf "$(BOLD)$(CYAN)Dependencies:$(RESET)\n"
	@printf "$(GREEN)%-20s$(RESET) %s\n" "make deps" "- обновить зависимости"
	@printf "$(GREEN)%-20s$(RESET) %s\n" "make deps-tidy" "- очистить зависимости"
	@printf "$(GREEN)%-20s$(RESET) %s\n" "make deps-verify" "- проверить зависимости"
	@echo ""
	@printf "$(BOLD)$(CYAN)Container Management:$(RESET)\n"
	@printf "$(YELLOW)%-20s$(RESET) %s\n" "make rm-image" "- удалить образ"
	@printf "$(YELLOW)%-20s$(RESET) %s\n" "make rm-container" "- удалить контейнер"
	@printf "$(YELLOW)%-20s$(RESET) %s\n" "make rm-all" "- удалить всё"
	@printf "$(BLUE)%-20s$(RESET) %s\n" "make logs" "- показать логи"
	@printf "$(BLUE)%-20s$(RESET) %s\n" "make status" "- показать статус"
	@printf "$(BLUE)%-20s$(RESET) %s\n" "make stop" "- остановить контейнер"
	@printf "$(BLUE)%-20s$(RESET) %s\n" "make restart" "- перезапустить"
	@printf "$(BLUE)%-20s$(RESET) %s\n" "make shell" "- войти в контейнер (shell)"
	@printf "$(BLUE)%-20s$(RESET) %s\n" "make exec" "- выполнить команду в контейнере"
	@echo ""
	@printf "$(BOLD)$(CYAN)Cleanup:$(RESET)\n"
	@printf "$(YELLOW)%-20s$(RESET) %s\n" "make clean" "- очистить бинарники и кэш"
	@printf "$(YELLOW)%-20s$(RESET) %s\n" "make clean-logs" "- очистить логи"
	@printf "$(YELLOW)%-20s$(RESET) %s\n" "make clean-all" "- полная очистка"
	@echo ""
	@printf "$(BOLD)$(CYAN)Information:$(RESET)\n"
	@printf "$(BLUE)%-20s$(RESET) %s\n" "make version" "- показать версии инструментов"
	@printf "$(BLUE)%-20s$(RESET) %s\n" "make info" "- показать информацию о проекте"
	@printf "$(BLUE)%-20s$(RESET) %s\n" "make config-check" "- проверить конфигурацию"
	@echo ""
	@printf "$(CYAN)Подсказка:$(RESET) используй $(GREEN)make help-vars$(RESET) для списка переменных\n"

help-vars:
	@echo ""
	@printf "$(BOLD)$(CYAN)%-25s$(RESET) %s\n" "Переменные (shortcuts):" ""
	@printf "$(GREEN)%-25s$(RESET) %s\n" "NAME_C" "- имя контейнера [$(NAME_C)]"
	@printf "$(GREEN)%-25s$(RESET) %s\n" "PORT_C" "- основной порт [$(PORT_C)]"
	@printf "$(GREEN)%-25s$(RESET) %s\n" "MPORT_C" "- порт метрик [$(MPORT_C)]"
	@printf "$(GREEN)%-25s$(RESET) %s\n" "NET_C" "- сеть [$(NET_C)]"
	@printf "$(YELLOW)%-25s$(RESET) %s\n" "NOCACHE_C" "- без кэша [$(NOCACHE_C)]"
	@printf "$(YELLOW)%-25s$(RESET) %s\n" "CACHEFROM_C" "- источник кэша [$(CACHEFROM_C)]"
	@printf "$(BLUE)%-25s$(RESET) %s\n" "LOGS_C" "- путь к логам [$(LOGS_C)]"
	@echo ""
	@printf "$(CYAN)Примеры:$(RESET)\n"
	@printf "  $(GREEN)make run-dev PORT_C=3000 NET_C=mynet$(RESET)\n"
	@printf "  $(GREEN)make build-dev NOCACHE_C=true$(RESET)\n"
	@printf "  $(GREEN)make prod-run NAME_C=app PORT_C=80$(RESET)\n"

# =============================================================================
# Shortcuts (алиасы для быстрого доступа)
# =============================================================================

h: help
hv: help-vars
tu: test-unit
ti: test-integration
ta: test-all
bd: build-dev
rd: run-dev
bl: build-local
rl: run-local
pb: prod-build
pr: prod-run
rmi: rm-image
rmc: rm-container
ra: rm-all
l: logs
s: status
st: stop
rs: restart
dt: deps-tidy
dv: deps-verify
du: deps-update
sw: swagger
v: version
i: info

# =============================================================================
# Tests
# =============================================================================

test-unit:
	go test ./... -short -v

test-integration:
	go test ./... -tags=integration -v -parallel=1

test-all:
	$(MAKE) test-unit && $(MAKE) test-integration

# =============================================================================
# Dev Application
# =============================================================================

build-dev:
	docker build \
		--target dev \
		$(container_build_args) \
		$(CACHE_OPT) \
		-t $(container_name):dev .

run-dev:
	docker run -d \
		--name $(container_name) \
		-p $(container_port):$(container_port) \
		-p $(container_metrics_port):$(container_metrics_port) \
		$(if $(wildcard .env),--env-file .env,) \
		-v $(logs_volume):/app/logs \
		--network $(container_network) \
		$(container_name):dev

# =============================================================================
# Prod Application
# =============================================================================

prod-build:
	docker build \
		--target prod \
		$(container_build_args) \
		$(CACHE_OPT) \
		-t $(container_name):prod .

prod-run:
	docker run -d \
		--name $(container_name) \
		--restart unless-stopped \
		-p $(container_port):$(container_port) \
		-p $(container_metrics_port):$(container_metrics_port) \
		$(if $(wildcard .env),--env-file .env,) \
		-v $(logs_volume):/app/logs \
		--network $(container_network) \
		$(container_name):prod

# =============================================================================
# Common Commands
# =============================================================================

rm-image:
	docker rmi $(container_name):dev $(container_name):prod || true

rm-container:
	docker rm -f $(container_name) || true

rm-all: stop rm-container rm-image

logs:
	docker logs -f $(container_name)

status:
	docker ps -a | grep $(container_name) || echo "Container $(container_name) not found"

stop:
	docker stop $(container_name) 2>/dev/null || true
	docker rm $(container_name) 2>/dev/null || true

restart: stop run-dev

run-container:
	docker run -d \
		--name $(container_name) \
		-p $(container_port):$(container_port) \
		-p $(container_metrics_port):$(container_metrics_port) \
		$(if $(wildcard .env),--env-file .env,) \
		-v $(logs_volume):/app/logs \
		--network $(container_network) \
		$(container_name):dev

# =============================================================================
# Dependencies Management
# =============================================================================

deps:
	@echo "Обновление зависимостей..."
	go mod download
	go mod tidy

deps-tidy:
	@echo "Очистка и обновление зависимостей..."
	go mod tidy

deps-download:
	@echo "Загрузка зависимостей..."
	go mod download

deps-verify:
	@echo "Проверка зависимостей..."
	go mod verify

deps-update:
	@echo "Обновление всех зависимостей..."
	go get -u ./...
	go mod tidy

# =============================================================================
# Code Quality
# =============================================================================

fmt:
	@echo "Форматирование кода..."
	go fmt ./...

vet:
	@echo "Проверка кода (go vet)..."
	go vet ./...

lint:
	@echo "Проверка кода линтером..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint не установлен. Установите: https://golangci-lint.run/usage/install/"; \
		exit 1; \
	fi

check: fmt vet test-unit
	@echo "Проверка кода завершена"

pre-commit: fmt vet test-unit swagger
	@echo "Предкоммитная проверка завершена"

# =============================================================================
# Swagger Documentation
# =============================================================================

swagger:
	@echo "Генерация Swagger документации..."
	@if command -v swag >/dev/null 2>&1; then \
		swag init -g cmd/api/main.go -o $(SWAGGER_DIR) --parseDependency --parseInternal; \
	else \
		echo "swag не установлен. Устанавливаю..."; \
		go install github.com/swaggo/swag/cmd/swag@v1.8.12; \
		swag init -g cmd/api/main.go -o $(SWAGGER_DIR) --parseDependency --parseInternal; \
	fi
	@echo "Swagger документация сгенерирована в $(SWAGGER_DIR)/"

swagger-clean:
	@echo "Очистка Swagger документации..."
	rm -rf $(SWAGGER_DIR)/*.go $(SWAGGER_DIR)/*.json $(SWAGGER_DIR)/*.yaml

# =============================================================================
# Local Build & Run
# =============================================================================

build-local:
	@echo "Сборка локального бинарника..."
	@mkdir -p bin
	CGO_ENABLED=1 go build -ldflags="-s -w" -o bin/$(BINARY_NAME) ./cmd/api
	@echo "Бинарник собран: bin/$(BINARY_NAME)"

run-local:
	@echo "Запуск приложения локально..."
	@if [ ! -f config.yaml ]; then \
		echo "Ошибка: config.yaml не найден. Создайте его из config.example.yaml"; \
		exit 1; \
	fi
	@if [ ! -f bin/$(BINARY_NAME) ]; then \
		echo "Бинарник не найден. Собираю..."; \
		$(MAKE) build-local; \
	fi
	./bin/$(BINARY_NAME)

# =============================================================================
# Cleanup
# =============================================================================

clean:
	@echo "Очистка бинарников и кэша..."
	rm -rf bin/
	go clean -cache -testcache -modcache
	@echo "Очистка завершена"

clean-logs:
	@echo "Очистка логов..."
	rm -rf $(LOGS_C)/*
	@echo "Логи очищены"

clean-all: clean clean-logs swagger-clean
	@echo "Полная очистка завершена"

# =============================================================================
# Container Utilities
# =============================================================================

shell:
	@echo "Вход в контейнер..."
	@docker exec -it $(container_name) /bin/sh || docker exec -it $(container_name) /bin/bash

exec:
	@if [ -z "$(CMD)" ]; then \
		echo "Использование: make exec CMD='команда'"; \
		echo "Пример: make exec CMD='ls -la'"; \
		exit 1; \
	fi
	@docker exec -it $(container_name) $(CMD)

# =============================================================================
# Information
# =============================================================================

version:
	@echo "Версия Go:"
	@go version
	@echo ""
	@echo "Версия Docker:"
	@docker --version || echo "Docker не установлен"
	@echo ""
	@if command -v swag >/dev/null 2>&1; then \
		echo "Версия Swagger: $$(swag version)"; \
	else \
		echo "Swagger не установлен"; \
	fi

info:
	@echo "$(BOLD)$(CYAN)Информация о проекте:$(RESET)"
	@echo "Имя контейнера: $(container_name)"
	@echo "HTTP порт: $(container_port)"
	@echo "Метрики порт: $(container_metrics_port)"
	@echo "Сеть: $(container_network)"
	@echo "Логи: $(logs_volume)"
	@echo "Бинарник: $(BINARY_NAME)"
	@echo "Swagger директория: $(SWAGGER_DIR)"
	@echo ""
	@echo "$(BOLD)$(CYAN)Статус контейнера:$(RESET)"
	@docker ps -a --filter name=$(container_name) --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" || echo "Контейнер не найден"
	@echo ""
	@echo "$(BOLD)$(CYAN)Статус образов:$(RESET)"
	@docker images --filter reference=$(container_name) --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}" || echo "Образы не найдены"

config-check:
	@echo "Проверка конфигурации..."
	@if [ ! -f config.yaml ]; then \
		echo "$(YELLOW)Предупреждение:$(RESET) config.yaml не найден"; \
		echo "Создайте его из config.example.yaml: $(GREEN)cp config.example.yaml config.yaml$(RESET)"; \
		exit 1; \
	fi
	@if [ ! -f config.example.yaml ]; then \
		echo "$(YELLOW)Предупреждение:$(RESET) config.example.yaml не найден"; \
	fi
	@if [ -f .env ]; then \
		echo "$(GREEN)✓$(RESET) .env файл найден"; \
	else \
		echo "$(YELLOW)ℹ$(RESET) .env файл не найден (необязательно для dev режима)"; \
	fi
	@echo "$(GREEN)✓$(RESET) Конфигурация проверена"
