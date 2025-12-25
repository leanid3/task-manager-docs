service := llm-worker
DB_NAME := llm_worker
DB_USER := postgres
DB_PASSWORD := postgres
DB_HOST := postgres-${service}
DB_PORT := 5432
DB_SSLMODE := disable
database := "postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)"
migrate_dir := /migrations
migrate_files := database/${service}/migrations/$(version)_*.up.sql database/${service}/migrations/$(version)_*.down.sql

##snipers:
mh: make migrate-help 
mc: make migrate-create
md: make migrate-delete
mu: make migrate-up
md: make migrate-down
ms: make migrate-status
mv: make migrate-version
mf: make migrate-force

migrate-help:
	@echo "Usage: make migrate-create name=<название_версии> version=<номер_версии>"
	@echo "Usage: make migrate-delete version=<номер_версии>"
	@echo "Usage: make migrate-up"
	@echo "Usage: make migrate-down"
	@echo "Usage: make migrate-status"
	@echo "Usage: make migrate-version"
	@echo "Usage: make migrate-force version=<номер_версии>"
	@echo "Usage: make db-psql query=<запрос>"
##Миграции
migrate-create:
	docker-compose run --rm ${service}-migrate create -ext sql -dir $(migrate_dir) -seq $(name)

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
	docker-compose run --rm ${service}-migrate -path $(migrate_dir) -database $(database) up

migrate-down:
	docker-compose run --rm ${service}-migrate -path $(migrate_dir) -database $(database) down

migrate-status:
	docker-compose run --rm ${service}-migrate -path $(migrate_dir) -database $(database) version

migrate-version:
	docker-compose run --rm ${service}-migrate -path $(migrate_dir) -database $(database) version

migrate-force:
	docker-compose run --rm ${service}-migrate -path $(migrate_dir) -database $(database) force $(version)



##Прямой доступ к бд
db-psql:
	docker-compose exec postgres-${service} psql -U $(DB_USER) -d $(DB_NAME) -c"$(query)"

