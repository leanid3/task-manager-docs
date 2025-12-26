.PHONY: help help-vars h hv tu ti ta bd rd pb pr rm rc ra
.PHONY: test-unit test-integration test-all build-dev run-dev
.PHONY: prod-build prod-run rm-image rm-container rm-all
.PHONY: logs status stop restart run-container

# =============================================================================
# Variables Configuration
# =============================================================================

# Default values (можно переопределить через CLI или окружение)
NAME_C ?= task-manager-app
PORT_C ?= 8080
MPORT_C ?= 9090
NET_C ?= bridge
NOCACHE ?= false
CACHEFROM_C ?= task-manager:dev
LOGS_C ?= ./logs

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
	@printf "$(GREEN)%-20s$(RESET) %s\n" "make build-dev" "- собрать образ в dev среде"
	@printf "$(GREEN)%-20s$(RESET) %s\n" "make run-dev" "- запустить контейнер в dev среде"
	@printf "$(GREEN)%-20s$(RESET) %s\n" "make prod-build" "- собрать образ в prod среде"
	@printf "$(GREEN)%-20s$(RESET) %s\n" "make prod-run" "- запустить контейнер в prod среде"
	@printf "$(YELLOW)%-20s$(RESET) %s\n" "make rm-image" "- удалить образ"
	@printf "$(YELLOW)%-20s$(RESET) %s\n" "make rm-container" "- удалить контейнер"
	@printf "$(YELLOW)%-20s$(RESET) %s\n" "make rm-all" "- удалить всё"
	@printf "$(BLUE)%-20s$(RESET) %s\n" "make logs" "- показать логи"
	@printf "$(BLUE)%-20s$(RESET) %s\n" "make status" "- показать статус"
	@printf "$(BLUE)%-20s$(RESET) %s\n" "make stop" "- остановить контейнер"
	@printf "$(BLUE)%-20s$(RESET) %s\n" "make restart" "- перезапустить"
	@echo ""
	@printf "$(CYAN)Подсказка:$(RESET) используй $(GREEN)make help-vars$(RESET) для списка переменных\n"

help-vars:
	@echo ""
	@printf "$(BOLD)$(CYAN)%-25s$(RESET) %s\n" "Переменные (shortcuts):" ""
	@printf "$(GREEN)%-25s$(RESET) %s\n" "NAME_C" "- имя контейнера [$(NAME_C)]"
	@printf "$(GREEN)%-25s$(RESET) %s\n" "PORT_C" "- основной порт [$(PORT_C)]"
	@printf "$(GREEN)%-25s$(RESET) %s\n" "MPORT_C" "- порт метрик [$(MPORT_C)]"
	@printf "$(GREEN)%-25s$(RESET) %s\n" "NET_C" "- сеть [$(NET_C)]"
	@printf "$(YELLOW)%-25s$(RESET) %s\n" "NOCACHE_C" "- без кэша [$(NOCACHE)]"
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
pb: prod-build
pr: prod-run
rmi: rm-image
rmc: rm-container
ra: rm-all
l: logs
s: status
st: stop
rs: restart

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
