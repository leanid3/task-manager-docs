service := task-manager
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
	@echo "Usage: make migrate-create service=<service_name> name=<migration_name>" - создание новой миграции
	@echo "Usage: make migrate-delete version=<номер_версии>" - удаление миграции по версии
	@echo "Usage: make migrate-up" - выполнение миграции всех версий
	@echo "Usage: make migrate-down" - откатить миграцию до прошлой версии
	@echo "Usage: make migrate-status" - узнать статус миграций
	@echo "Usage: make migrate-version" - узнать версию миграции
	@echo "Usage: make migrate-force version=<номер_версии>" - принудительно установить весрию миграции
	@echo "Usage: make db-psql query=<запрос>" - прямой доступ к бд
##Миграции
migrate-create:
		@if [ -z "$(name)" ]; then \
		echo "Usage: make migrate-create service=<service_name> name=<migration_name>"; \
		exit 1; \
	fi
	docker-compose run --rm $(service)-migrate \
		create -ext sql -dir $(migrate_dir) -seq $(name)

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
	@rm -f $(migrate_files)
	@echo "Файлы миграции $(version) удалены"

migrate-up:
	docker-compose run --rm ${service}-migrate -path $(migrate_dir) -database $(database) up

migrate-down:
	docker-compose run --rm ${service}-migrate -path $(migrate_dir) -database $(database) down


migrate-force:
	docker-compose run --rm ${service}-migrate -path $(migrate_dir) -database $(database) force $(version)


## Получить информацию о миграциях
migrate-status:
	docker-compose run --rm ${service}-migrate -path $(migrate_dir) -database $(database) version
migrate-version:
	docker-compose run --rm ${service}-migrate -path $(migrate_dir) -database $(database) version


##Прямой доступ к бд
db-psql:
	docker-compose exec postgres-${service} psql -U $(DB_USER) -d $(DB_NAME) -c"$(query)"

