APP_SERVICE=app

.PHONY: up down build logs reset-db

## Полный запуск проекта 
up:
	docker-compose up --build

## Остановка контейнеров
down:
	docker-compose down

build:
	docker-compose build $(APP_SERVICE)

## Логи приложения
logs:
	docker-compose logs -f $(APP_SERVICE)

## Полный сброс БД и запуск заново
reset-db:
	docker-compose down -v
	docker-compose up --build