# Integration Tests Structure

## Overview
Интеграционные тесты организованы по доменной модели для лучшей читаемости и поддержки.

## Directory Structure
```
tests/integration/
├── setup_test.go              # Глобальная инициализация тестовой среды
├── helpers.go                 # Общие вспомогательные функции
├── auth_ratelimit_test.go    # Rate limiting для auth операций
├── auth_session_test.go      # Управление сессиями (будущее)
├── auth_validation_test.go   # Валидация токенов (будущее)
├── user_crud_test.go         # CRUD операции с пользователями
└── user_privacy_test.go      # Privacy rules для пользователей (будущее)
```

**Важно**: Из-за ограничений Go (пакеты в подпапках изолированы), используется плоская структура с префиксами доменов в именах файлов.

## File Descriptions

### Infrastructure Layer
- **`setup_test.go`** - Инициализация тестового окружения:
  - Запуск PostgreSQL в Docker контейнере
  - Выполнение миграций БД
  - Создание базовых тестовых пользователей
  - Инициализация i18n и логгера

- **`helpers.go`** - Общие helper функции:
  - `withUserContext()` - создание контекста с пользователем
  - `createTestUser()` - создание тестового пользователя
  - `generateTestTenantID()` - генерация тестового tenant ID
  - Другие утилиты для тестов

### Domain Tests

#### Auth Domain (префикс `auth_`)
- **`auth_ratelimit_test.go`** - Тестирование rate limiting:
  - Ограничения на количество попыток логина
  - Блокировка после превышения лимита
  - Сброс счетчиков после успешного входа
  - Работа с разными типами идентификаторов (email, IP)

#### User Domain (префикс `user_`)
- **`user_crud_test.go`** - Базовые операции с пользователями:
  - Создание пользователей
  - Поиск по ID и email
  - Обновление данных
  - Проверка уникальности email

## Naming Conventions

### Test Files
- Формат: `{domain}_{feature}_test.go`
- Примеры: `auth_session_test.go`, `user_privacy_test.go`, `auth_ratelimit_test.go`

### Test Functions
- Главная функция: `Test{Domain}{Feature}` (e.g., `TestAuthRateLimit`)
- Подтесты: `test{Feature}{Aspect}` (e.g., `testRateLimitExceeded`)

### Helper Functions
- Создание данных: `create{Entity}` (e.g., `createTestUser`)
- Контекст: `with{Context}` (e.g., `withUserContext`)
- Утилиты: `{action}{Entity}` (e.g., `generateTestTenantID`)

## Running Tests

### All Integration Tests
```bash
make test-integration
```

### Specific Domain Tests
```bash
# Auth domain tests (все файлы с префиксом auth_)
go test -tags=integration ./tests/integration -run "TestAuth" -v

# User domain tests (все файлы с префиксом user_)
go test -tags=integration ./tests/integration -run "TestUser" -v
```

### Single Test File
```bash
go test -tags=integration ./tests/integration -run TestAuthRateLimit -v
```

### With Coverage
```bash
go test -tags=integration ./tests/integration/... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Best Practices

### 1. Test Isolation
- Каждый тест должен быть независимым
- Используйте `TestHelper` для транзакционной изоляции
- Откатывайте изменения после каждого теста

### 2. Test Data
- Используйте уникальные email адреса (с timestamp)
- Создавайте минимально необходимый набор данных
- Очищайте ресурсы в defer функциях

### 3. Context Usage
- Всегда используйте `privacy.WithSystemContext()` для обхода privacy rules в setup
- Используйте `withUserContext()` для тестирования с правами пользователя
- Добавляйте timeout к контексту для предотвращения зависаний

### 4. Error Handling
- Всегда проверяйте ошибки в setup функциях
- Используйте `require.NoError()` для критичных операций
- Используйте `assert` для некритичных проверок

### 5. Naming
- Давайте понятные имена тестам, отражающие что тестируется
- Группируйте связанные тесты через t.Run()
- Добавляйте комментарии к сложной логике

## Adding New Tests

### 1. Determine Domain
Определите, к какому домену относится функциональность:
- Authentication/Authorization → префикс `auth_`
- User management → префикс `user_`
- Future domains → используйте соответствующий префикс

### 2. Create Test File
```go
//go:build integration

package integration

import (
    "testing"
    // ... imports
)

func TestDomainFeature(t *testing.T) {
    helper := NewTestHelper(t)
    defer helper.Rollback()
    
    ctx, cancel := context.WithTimeout(helper.GetContext(), 30*time.Second)
    defer cancel()
    
    t.Run("SubTest", func(t *testing.T) {
        // test implementation
    })
}
```

### 3. Use Existing Helpers
Используйте существующие helper функции из `helpers.go` и `setup_test.go`

### 4. Document Complex Logic
Добавляйте комментарии для сложной бизнес-логики или неочевидных проверок

## Troubleshooting

### Common Issues

1. **"sql: database is closed"**
   - Используйте `TestHelper` вместо прямого клиента
   - Не закрывайте глобальное соединение

2. **"current transaction is aborted"**
   - Проверяйте ошибки в setup функциях
   - Используйте отдельный `TestHelper` для каждого подтеста

3. **"duplicate key value"**
   - Используйте `time.Now().UnixNano()` для уникальности
   - Не используйте статические email адреса

4. **Privacy rule violations**
   - Используйте `privacy.WithSystemContext()` в setup
   - Правильно устанавливайте user context для тестов

## Future Improvements

### Planned Tests
- [ ] `auth/session_test.go` - Управление сессиями
- [ ] `auth/validation_test.go` - Валидация токенов и permissions
- [ ] `user/privacy_test.go` - Privacy rules для пользователей
- [ ] `auth/oauth_test.go` - OAuth интеграции (when implemented)

### Infrastructure
- [ ] Добавить benchmark тесты для критичных операций
- [ ] Интеграция с CI/CD для автоматического запуска
- [ ] Параллельный запуск тестов где возможно
- [ ] Метрики покрытия кода по доменам