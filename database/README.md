# Database Package

Клиент для работы с PostgreSQL через Ent ORM с поддержкой:
- Разделения клиентов для чтения (Query) и записи (Mutation)
- Многоуровневого кэширования (контекстное + Redis с изоляцией по тенантам)
- Автоматической инвалидации кэша при мутациях
- Версионирования кэша для тенантов

## Архитектура

### Основные компоненты

- **Два подключения**: отдельные DSN для чтения (`DB_QUERY_HOST`) и записи (`DB_MUTATION_HOST`)
- **Многоуровневое кэширование**:
  - Контекстное кэширование (entcache.ContextLevel) - дедупликация запросов в рамках одного HTTP-запроса
  - Redis кэш с изоляцией по тенантам - кэширование между запросами
- **Автоматическая инвалидация**: при мутациях автоматически инкрементируется версия кэша тенанта
- **Глобальный клиент**: один экземпляр на процесс, инициализируется лениво при первом запросе

## Переменные окружения

### Основные параметры

| Переменная | Описание | Значение по умолчанию |
|------------|----------|----------------------|
| `DB_QUERY_HOST` | Хост для чтения | `localhost` |
| `DB_QUERY_PORT` | Порт для чтения | `5432` |
| `DB_MUTATION_HOST` | Хост для записи | `localhost` |
| `DB_MUTATION_PORT` | Порт для записи | `5432` |
| `DB_USER` | Пользователь БД | `postgres` |
| `DB_PASSWORD` | Пароль пользователя | - |
| `DB_NAME` | Имя базы данных | `postgres` |
| `DB_SSLMODE` | Режим SSL соединения | `disable` |
| `DB_SCHEMA` | Схема по умолчанию (search_path) | `app` |

### Отладка и кэширование

| Переменная     | Описание | Значение по умолчанию |
|----------------|----------|----------------------|
| `DEBUG_DB`     | Логирование SQL запросов | `false` |
| `ENABLE_DB_CACHE` | Включить кэширование (контекст + Redis) | `true` |
| `DB_CACHE_TTL` | TTL кэша в секундах | `300` (5 минут) |

### Настройки пула соединений

| Переменная | Описание | Значение по умолчанию |
|------------|----------|----------------------|
| `DB_MAX_OPEN_CONNS` | Максимальное количество открытых соединений | `10` |
| `DB_MAX_IDLE_CONNS` | Максимальное количество idle соединений | `5` |
| `DB_CONN_MAX_LIFETIME` | Максимальное время жизни соединения (секунды) | `300` (5 минут) |
| `DB_CONN_MAX_IDLE_TIME` | Максимальное время простоя соединения (секунды) | `60` (1 минута) |

## Использование

### Инициализация

```go
import (
    "context"
    "main/middleware"
)

// Вариант 1: Явная инициализация при старте
if err := middleware.InitDatabaseClient(context.Background()); err != nil {
    log.Fatal(err)
}

// Вариант 2: Ленивая инициализация через middleware
// DatabaseMiddleware автоматически инициализирует клиент при первом запросе
r.Use(middleware.DatabaseMiddleware)
```

### Graceful Shutdown

```go
// В main.go при обработке сигнала завершения
<-shutdown
utils.Logger.Info("Shutdown signal received...")

// Закрываем соединения с БД
if err := middleware.CloseDatabaseClient(); err != nil {
    utils.Logger.Error("Database shutdown error", zap.Error(err))
} else {
    utils.Logger.Info("Database shutdown complete")
}
```

### Роутер и GraphQL

```go
r := chi.NewRouter()
r.Use(middleware.RequestIDMiddleware)

// Создаем GraphQL сервер один раз на процесс
db := middleware.GetDatabaseClient()
if db == nil { log.Fatal("db not init") }
graphqlServer := server.NewGraphQLServer(bundle, db)

r.Group(func(r chi.Router) {
    r.Use(middleware.HeadersMiddleware)
    r.Use(middleware.LocalAuthMiddleware())
    r.Handle("/query", graphqlServer)
})
```

### Управление кэшированием

```go
// Включить контекстное кэширование для запроса
ctx = database.EnableContextCache(ctx)

// Пропустить кэширование (для мутаций)
ctx = database.SkipCache(ctx)

// Проверка режима отладки
if database.IsDebugDB() {
    log.Println("Database debug mode enabled")
}
```

### Работа с транзакциями

```go
db := middleware.GetDatabaseClient()
err := db.WithTx(ctx, func(tx *ent.Tx) error {
    // Выполнение операций в транзакции
    user, err := tx.User.Create().
        SetName("John").
        Save(ctx)
    if err != nil {
        return err // Автоматический rollback
    }
    
    // Другие операции...
    return nil // Автоматический commit
})
```

## Миграции

```go
ctx := context.Background()
client, err := database.NewClient(ctx, nil)
if err != nil { log.Fatal(err) }
defer client.Close()

if err := client.Mutation().Schema.Create(ctx); err != nil {
    log.Fatal(err)
}
```

## Особенности реализации

### Разделение Read/Write
- **Query client**: используется для всех операций чтения, включает кэширование
- **Mutation client**: используется для операций записи, без кэширования, но с хуком инвалидации

### Управление пулом соединений
- **Автоматическое управление**: Go поддерживает внутренний пул соединений
- **Совместимость с прокси**: настройки пула оптимизированы для работы с PgBouncer/pgpool
- **Lifecycle соединений**:
  - Соединения переиспользуются между запросами
  - Idle соединения закрываются через `DB_CONN_MAX_IDLE_TIME`
  - Все соединения пересоздаются через `DB_CONN_MAX_LIFETIME`
  - При shutdown все соединения корректно закрываются

### Многоуровневое кэширование
1. **Context-level cache**: дедупликация запросов в рамках одного HTTP-запроса
2. **Redis cache**: кэширование между запросами с изоляцией по тенантам
   - Ключи с префиксом `entcache:v2:tenant:{tenant_id}:v{version}:`
   - Автоматическое версионирование при мутациях
   - TTL по умолчанию 5 минут

### Изоляция тенантов
- Кэш полностью изолирован между тенантами
- При мутации инвалидируется только кэш конкретного тенанта
- Глобальные операции (без тенанта) используют префикс `global`

### Автоматическая инвалидация
- При операциях Create/Update/Delete автоматически инкрементируется версия кэша тенанта
- Инвалидация происходит асинхронно в фоне (timeout 5 секунд)
- Старые версии кэша автоматически становятся недействительными

## Зависимости

### Внешние сервисы
- PostgreSQL 12+ (основная БД)
- Redis (опционально, для кэширования между запросами)

### Go пакеты
- `entgo.io/ent` - ORM
- `ariga.io/entcache` - кэширование для Ent
- `github.com/jackc/pgx/v5` - PostgreSQL драйвер
- `github.com/go-redis/redis/v8` - Redis клиент

## Мониторинг и отладка

### Логирование
- SQL запросы логируются при `DEBUG_DB=true`
- Ошибки инвалидации кэша логируются с уровнем ERROR
- Успешная инициализация логируется с уровнем INFO
- Закрытие соединений при shutdown логируется

### Проверка утечек соединений
```sql
-- Проверка активных соединений в PostgreSQL
SELECT pid, usename, application_name, client_addr, state 
FROM pg_stat_activity 
WHERE datname = 'your_database';

-- Проверка через PgBouncer (если используется)
SHOW POOLS;
SHOW CLIENTS;
```

### Метрики кэша
- Версия кэша тенанта хранится в Redis: `entcache:v2:tenant:{tenant_id}:version`
- При каждой мутации версия инкрементируется
- Можно отслеживать частоту инвалидаций по изменению версии

## Рекомендации для Production

### При использовании PgBouncer/pgpool

1. **Настройте пул соединений для вашей нагрузки**:
   ```bash
   # Для высоконагруженных сервисов
   DB_MAX_OPEN_CONNS=20
   DB_MAX_IDLE_CONNS=5
   DB_CONN_MAX_LIFETIME=300  # 5 минут
   DB_CONN_MAX_IDLE_TIME=60  # 1 минута
   
   # Для сервисов с низкой нагрузкой
   DB_MAX_OPEN_CONNS=5
   DB_MAX_IDLE_CONNS=2
   DB_CONN_MAX_LIFETIME=600  # 10 минут
   DB_CONN_MAX_IDLE_TIME=120 # 2 минуты
   ```

2. **Настройте PgBouncer в режиме session или transaction**:
   - `session` - для долгих транзакций
   - `transaction` - для коротких запросов (рекомендуется)

3. **Мониторинг**:
   - Отслеживайте количество соединений в пуле
   - Проверяйте, что соединения корректно закрываются при рестарте
   - Используйте метрики прокси для оптимизации настроек