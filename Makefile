DB_USER := postgres
DB_PASSWORD := postgres
DB_HOST := postgres
DB_PORT := 5432
DB_NAME := document_flow
DB_SSLMODE := disable
database := "postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)"
migrate_dir := /migrations
migrate_files := database/task-manager/migrations/$(version)_*.up.sql database/task-manager/migrations/$(version)_*.down.sql

##Миграции
migrate-create:
	docker-compose run --rm migrate create -ext sql -dir $(migrate_dir) -seq $(name)

migrate-delete:
	@if [ -z "$(version)" ]; then \
		echo "Usage: make migrate-delete version=<номер_версии>"; \
		echo "Example: make migrate-delete version=000005"; \
		exit 1; \
	fi

	@if [ ! -e $(word 1,$(migrate_files)) ] && [ ! -e $(word 2,$(migrate_files)) ]; then \
		echo "Файлы миграции версии $(version) не найдены"; \
		exit 1; \
	fi
	@echo "Удаление файлов миграции версии $(version)..."
	@echo "--------------------------------"
	@rm -f $(migrate_files)
	@echo "Файлы миграции $(version) удалены"

migrate-up:
	docker-compose run --rm migrate -path $(migrate_dir) -database $(database) up

migrate-down:
	docker-compose run --rm migrate -path $(migrate_dir) -database $(database) down

migrate-status:
	docker-compose run --rm migrate -path $(migrate_dir) -database $(database) version

migrate-version:
	docker-compose run --rm migrate -path $(migrate_dir) -database $(database) version

migrate-force:
	docker-compose run --rm migrate -path $(migrate_dir) -database $(database) force $(version)



##Прямой доступ к бд
db-psql:
	docker-compose exec postgres psql -U $(DB_USER) -d $(DB_NAME) -c"$(query)"