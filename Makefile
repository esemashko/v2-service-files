# Makefile для tairo-core-backend

# Переменные
GO_CMD=go
GO_TEST=go test
GO_BUILD=go build
GO_CLEAN=go clean
GO_GET=go get
GO_MOD=go mod
DOCKER_CMD=docker
DOCKER_COMPOSE=docker compose

# Цвета для вывода
RED=\033[0;31m
GREEN=\033[0;32m
YELLOW=\033[0;33m
NC=\033[0m # No Color

# Помощь
.PHONY: help
help: ## Показать помощь
	@echo "Доступные команды:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Установка зависимостей
.PHONY: deps
deps: ## Установить зависимости
	@echo "$(YELLOW)Установка зависимостей...$(NC)"
	$(GO_MOD) tidy
	$(GO_MOD) download

# Обновление зависимостей
.PHONY: deps-update
deps-update: ## Обновить зависимости
	@echo "$(YELLOW)Обновление зависимостей...$(NC)"
	$(GO_GET) -u ./...
	$(GO_MOD) tidy

# Сборка
.PHONY: build
build: ## Собрать приложение
	@echo "$(YELLOW)Сборка приложения...$(NC)"
	$(GO_BUILD) -o bin/main .

# Очистка
.PHONY: clean
clean: ## Очистить билды
	@echo "$(YELLOW)Очистка...$(NC)"
	$(GO_CLEAN)
	rm -rf bin/

# Генерация кода
.PHONY: generate
generate: ## Генерация кода (ent + gqlgen)
	@echo "$(YELLOW)Генерация ENT кода...$(NC)"
	$(GO_CMD) run entgo.io/ent/cmd/ent generate ./ent/schema
	@echo "$(YELLOW)Генерация GraphQL кода...$(NC)"
	$(GO_CMD) run github.com/99designs/gqlgen generate

# Тесты
.PHONY: test
test: ## Запустить все тесты
	@echo "$(YELLOW)Запуск всех тестов...$(NC)"
	$(GO_TEST) -v ./...

.PHONY: test-unit
test-unit: ## Запустить юнит тесты
	@echo "$(YELLOW)Запуск юнит тестов...$(NC)"
	$(GO_TEST) -v -short ./...

.PHONY: test-integration
test-integration: ## Запустить интеграционные тесты
	@echo "$(YELLOW)Запуск интеграционных тестов...$(NC)"
	$(GO_TEST) -tags integration -v -timeout=10m main/tests/integration

.PHONY: test-failures
test-failures: ## Найти упавшие интеграционные тесты
	@echo "$(YELLOW)Поиск упавших тестов...$(NC)"
	@./scripts/test-failures.sh

.PHONY: test-single
test-single: ## Запустить один тест (usage: make test-single TEST=TestName)
	@echo "$(YELLOW)Запуск теста $(TEST)...$(NC)"
	@./scripts/test-single.sh "$(TEST)"

.PHONY: test-auth
test-auth: ## Запустить тесты домена auth (с префиксом auth_)
	@echo "$(YELLOW)Запуск тестов домена auth...$(NC)"
	$(GO_TEST) -tags integration -v -timeout=10m main/tests/integration -run "TestAuth"

.PHONY: test-user
test-user: ## Запустить тесты домена user (с префиксом user_)
	@echo "$(YELLOW)Запуск тестов домена user...$(NC)"
	$(GO_TEST) -tags integration -v -timeout=10m main/tests/integration -run "TestUser"

.PHONY: test-notifications
test-notifications: ## Запустить тесты системы уведомлений
	@echo "$(YELLOW)Запуск тестов системы уведомлений...$(NC)"
	$(GO_TEST) -v -timeout=5m ./tests/integration/... -run="NotificationSystem"

.PHONY: test-coverage
test-coverage: ## Запустить тесты с покрытием кода
	@echo "$(YELLOW)Запуск тестов с покрытием кода...$(NC)"
	$(GO_TEST) -v -coverprofile=coverage.out ./...
	$(GO_CMD) tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Отчет о покрытии сохранен в coverage.html$(NC)"

.PHONY: test-race
test-race: ## Запустить тесты с проверкой race conditions
	@echo "$(YELLOW)Запуск тестов с проверкой race conditions...$(NC)"
	$(GO_TEST) -v -race ./...

.PHONY: test-bench
test-bench: ## Запустить бенчмарки
	@echo "$(YELLOW)Запуск бенчмарков...$(NC)"
	$(GO_TEST) -v -bench=. -benchmem ./...

# Линтинг
.PHONY: lint
lint: ## Запустить линтеры
	@echo "$(YELLOW)Запуск линтеров...$(NC)"
	golangci-lint run

.PHONY: lint-fix
lint-fix: ## Исправить проблемы линтинга
	@echo "$(YELLOW)Исправление проблем линтинга...$(NC)"
	golangci-lint run --fix

# Форматирование кода
.PHONY: fmt
fmt: ## Форматировать код
	@echo "$(YELLOW)Форматирование кода...$(NC)"
	$(GO_CMD) fmt ./...

.PHONY: fmt-imports
fmt-imports: ## Исправить импорты
	@echo "$(YELLOW)Исправление импортов...$(NC)"
	goimports -w .

# Запуск приложения
.PHONY: run
run: ## Запустить приложение
	@echo "$(YELLOW)Запуск приложения...$(NC)"
	$(GO_CMD) run .

.PHONY: run-dev
run-dev: ## Запустить приложение в режиме разработки
	@echo "$(YELLOW)Запуск приложения в режиме разработки...$(NC)"
	DEBUG=true DEBUG_DB=true $(GO_CMD) run .

# Docker
.PHONY: docker-build
docker-build: ## Собрать Docker образ
	@echo "$(YELLOW)Сборка Docker образа...$(NC)"
	$(DOCKER_CMD) build -t tairo-core-backend .

.PHONY: docker-run
docker-run: ## Запустить приложение в Docker
	@echo "$(YELLOW)Запуск приложения в Docker...$(NC)"
	$(DOCKER_CMD) run -p 8080:8080 tairo-core-backend

# База данных
.PHONY: db-migrate
db-migrate: ## Выполнить миграции базы данных
	@echo "$(YELLOW)Выполнение миграций базы данных...$(NC)"
	$(GO_CMD) run . migrate

.PHONY: db-seed
db-seed: ## Заполнить базу данных seeds
	@echo "$(YELLOW)Заполнение базы данных seeds...$(NC)"
	$(GO_CMD) run . seed

.PHONY: db-reset
db-reset: ## Сбросить базу данных (пересоздать seeds)
	@echo "$(YELLOW)Сброс базы данных...$(NC)"
	$(GO_CMD) run . seed --reset

# Разработка
.PHONY: dev-setup
dev-setup: deps generate ## Настроить среду разработки
	@echo "$(GREEN)Среда разработки настроена!$(NC)"

.PHONY: dev-check
dev-check: fmt lint test-unit ## Проверить код перед коммитом
	@echo "$(GREEN)Проверка кода завершена!$(NC)"

# Утилиты
.PHONY: vendor
vendor: ## Создать vendor директорию
	@echo "$(YELLOW)Создание vendor директории...$(NC)"
	$(GO_MOD) vendor

.PHONY: tidy
tidy: ## Очистить go.mod и go.sum
	@echo "$(YELLOW)Очистка go.mod и go.sum...$(NC)"
	$(GO_MOD) tidy

# Профилирование
.PHONY: profile-cpu
profile-cpu: ## Запустить профилирование CPU
	@echo "$(YELLOW)Запуск профилирования CPU...$(NC)"
	$(GO_TEST) -cpuprofile=cpu.prof -bench=. ./...

.PHONY: profile-mem
profile-mem: ## Запустить профилирование памяти
	@echo "$(YELLOW)Запуск профилирования памяти...$(NC)"
	$(GO_TEST) -memprofile=mem.prof -bench=. ./...

# Проверка безопасности
.PHONY: security
security: ## Проверить безопасность кода
	@echo "$(YELLOW)Проверка безопасности кода...$(NC)"
	gosec ./...

# Документация
.PHONY: docs
docs: ## Генерировать документацию
	@echo "$(YELLOW)Генерация документации...$(NC)"
	godoc -http=:6060
	@echo "$(GREEN)Документация доступна по адресу: http://localhost:6060$(NC)"

# Установка инструментов разработки
.PHONY: install-tools
install-tools: ## Установить инструменты разработки
	@echo "$(YELLOW)Установка инструментов разработки...$(NC)"
	$(GO_GET) -u github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GO_GET) -u golang.org/x/tools/cmd/goimports@latest
	$(GO_GET) -u github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	$(GO_GET) -u github.com/go-delve/delve/cmd/dlv@latest
	@echo "$(GREEN)Инструменты разработки установлены!$(NC)"

# Отладка
.PHONY: debug
debug: ## Запустить отладчик
	@echo "$(YELLOW)Запуск отладчика...$(NC)"
	dlv debug

# Все задачи для CI/CD
.PHONY: ci
ci: deps generate fmt lint test-race test-coverage ## Запустить все проверки CI/CD
	@echo "$(GREEN)Все проверки CI/CD завершены!$(NC)"

# Проверить готовность к продакшену
.PHONY: production-ready
production-ready: ci security ## Проверить готовность к продакшену
	@echo "$(GREEN)Проект готов к продакшену!$(NC)"

# Помощь по командам тестирования
.PHONY: test-help
test-help: ## Показать помощь по командам тестирования
	@echo "$(YELLOW)Команды тестирования:$(NC)"
	@echo "  $(GREEN)make test$(NC)                 - Запустить все тесты"
	@echo "  $(GREEN)make test-unit$(NC)            - Запустить только юнит тесты"
	@echo "  $(GREEN)make test-integration$(NC)     - Запустить интеграционные тесты"
	@echo "  $(GREEN)make test-notifications$(NC)   - Запустить тесты системы уведомлений"
	@echo "  $(GREEN)make test-coverage$(NC)        - Запустить тесты с покрытием кода"
	@echo "  $(GREEN)make test-race$(NC)            - Запустить тесты с проверкой race conditions"
	@echo "  $(GREEN)make test-bench$(NC)           - Запустить бенчмарки"
	@echo ""
	@echo "$(YELLOW)Примеры:$(NC)"
	@echo "  $(GREEN)go test -v ./tests/integration/... -run='TestNotificationChannels'$(NC)"
	@echo "  $(GREEN)go test -v ./tests/integration/... -run='NotificationSystem'$(NC)"
	@echo "  $(GREEN)go test -v -timeout=10m ./tests/integration/...$(NC)"

# Анализ логов запросов
.PHONY: analyze-logs
analyze-logs: ## Анализировать логи запросов
	@echo "$(YELLOW)Анализ логов запросов...$(NC)"
	@echo "Используйте следующие команды:"
	@echo "  $(GREEN)make analyze-noisy-logs$(NC)   - Найти шумные логи"
	@echo "  $(GREEN)make analyze-n1$(NC)          - Найти N+1 проблемы"

# Анализ шумных логов
.PHONY: analyze-noisy-logs
analyze-noisy-logs: ## Найти самые частые (шумные) логи
	@echo "$(YELLOW)Анализ шумных логов...$(NC)"
	@go run ./tools/analyze_noisy_logs/main.go -dir query_logs -top 20 -min 10
	@echo ""
	@echo "$(YELLOW)Для сохранения списка игнорируемых сообщений:$(NC)"
	@echo "  $(GREEN)go run ./tools/analyze_noisy_logs/main.go -dir query_logs -output noisy_logs.txt$(NC)"

# Поиск N+1 проблем в логах
.PHONY: analyze-n1
analyze-n1: ## Найти N+1 проблемы в запросах
	@echo "$(YELLOW)Анализ N+1 проблем в логах...$(NC)"
	@go run ./tools/detect_n1/main.go query_logs

# По умолчанию - показать помощь
.DEFAULT_GOAL := help 