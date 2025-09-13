# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## CRITICAL: Development Server Management & Code Generation

**⚠️ ВАЖНО: НИКОГДА не запускайте сервер разработки (`make run-dev` или `go run .`) - приложение уже запущено разработчиком!**

**⚠️ ВАЖНО: НИКОГДА не запускайте генерацию кода (`make generate`, `go generate`, `go run ./ent/entc.go`) - только разработчик может это делать!**

Если нужно проверить изменения:
1. Убедитесь, что код скомпилируется: `go build -o /tmp/test main.go`
2. Проверьте логи в `/query_logs/` для отладки (логи не всегда доступны)
3. Попросите разработчика перезапустить сервер, если требуется

Если нужна генерация кода после изменения схем:
1. Внесите изменения в схемы (GraphQL, Ent)
2. **ПОПРОСИТЕ РАЗРАБОТЧИКА** выполнить `make generate`
3. НЕ пытайтесь запускать генерацию самостоятельно!

## Project Overview

Multi-tenant B2B SaaS Helpdesk System с Clean Architecture.

**Technology Stack:**
- Go 1.25.0, GraphQL API (gqlgen), Ent ORM, PostgreSQL, Redis, i18n

## Microservice Architecture

### Service Isolation
Приложение разбито на несколько изолированных контейнеров:
- **Сервис авторизации**: Хранит пользователей, отделы, руководителей отделов
- **Сервис хранения файлов**: Управляет загрузкой и хранением файлов
- **Сервис тикетов**: Управляет тикетами и связанными сущностями

**КРИТИЧЕСКИ ВАЖНО**:
- Сервисы НЕ общаются напрямую между собой
- Хранят только ссылки (UUID) на сущности из других сервисов
- НЕ имеют доступа к EDGE ребрам между сервисами
- Валидация и проверка связанных данных невозможна внутри сервиса

### Federation Context Access
Доступ к данным пользователя через расширение federation `"github.com/esemashko/v2-federation"`:

```go
import "github.com/esemashko/v2-federation"

// Получение данных текущего пользователя из контекста
userID := federation.GetUserID(ctx)                    // UUID пользователя
userRole := federation.GetUserRole(ctx)                // Роль пользователя
departmentIDs := federation.GetDepartmentIDs(ctx)      // Отделы пользователя
managedDeptIDs := federation.GetManagedDepartmentIDs(ctx) // Управляемые отделы
```

### Важные ограничения:
1. **Нет прямой валидации**: Невозможно проверить существование пользователя/отдела в момент создания тикета
2. **Только через GraphQL Federation**: Полные данные доступны только через Apollo Router
3. **Асинхронная согласованность**: Данные между сервисами могут быть временно несогласованны
4. **Используйте context данные**: Всегда полагайтесь на данные из federation context

**Essential Commands:**
```bash
# Development  
make run-dev       # Run with debug
make generate      # Regenerate after schema changes
make dev-check     # Format + lint + unit tests

# Testing
make test-unit        # Quick unit tests
make test-integration # Integration tests
make test-single TEST=TestName # Specific test
```

## Critical Development Guidelines

### 1. Logging
- **Always use utils.Logger**, NOT zap.Logger directly:
  ```go
  utils.Logger.Error("error message", zap.Error(err))
  utils.Logger.Info("info message", zap.String("key", "value"))
  utils.Logger.Debug("debug message")
  ```

### 2. Database Client Architecture and Resolver Pattern

**CRITICAL**: The application uses separate database clients for Query and Mutation operations for performance and caching optimization.

#### Database Client Setup
1. **Two separate clients**:
    - `db.Query()` - Read-only client with Redis caching enabled
    - `db.Mutation()` - Write client without caching but with cache invalidation hooks

2. **Client selection happens in server.go**:
   ```go
   // server.go automatically selects the right client based on operation type
   case ast.Query:
       entClient = db.Query()      // Uses cached read client
   case ast.Mutation, ast.Subscription:
       entClient = db.Mutation()   // Uses write client with invalidation hooks
   ```

3. **Resolver must use getClient(ctx)**:
   ```go
   // In resolver.go
   func (r *Resolver) getClient(ctx context.Context) *ent.Client {
       if client := ent.FromContext(ctx); client != nil {
           return client  // Returns the correct client from context
       }
       return r.client   // Fallback for tests
   }
   ```

#### Important for ALL Resolvers:
- **ALWAYS use `client := r.getClient(ctx)`** at the start of resolver methods
- **NEVER use `r.client` directly** - it always points to Query client
- This ensures:
    - Mutations use the write client with cache invalidation
    - Queries use the read client with caching
    - Cache invalidation hooks work correctly

### 3. Transaction Management Architecture

**CRITICAL**: Transactions ONLY at resolver layer, NEVER in services.

#### Rules:
1. **Resolvers**: Handle ALL transaction lifecycle
2. **Services**: NEVER create transactions - receive `*ent.Client` that may be transactional
3. **External operations**: Only in post-commit hooks

2. **Service layer**: NEVER creates transactions
    - Receives `*ent.Client` that may already be transactional
    - Works with the client as-is (no `client.Tx()` calls)
    - Can perform multiple operations using the same client

3. **External operations** (Redis, notifications, etc.) must be in post-commit hooks:
   ```go
   err = tx.OnCommit(func(next ent.Committer) ent.Committer {
       return ent.CommitFunc(func(ctx context.Context, clientTx *ent.Tx) error {
           err := next.Commit(ctx, clientTx)
           if err == nil {
               // External operations here
           }
           return err
       })
   })
   ```

#### Correct Patterns:

**✅ Resolver with transaction:**
```go
func (r *mutationResolver) CreateEntity(ctx context.Context, input model.Input) (*model.Response, error) {
    // ВАЖНО: Используем getClient для получения правильного клиента (mutation/query)
    client := r.getClient(ctx)
    
    // Start transaction in resolver
    tx, err := client.Tx(ctx)
    if err != nil {
        return &model.Response{Success: false, Message: "Transaction failed"}, nil
    }
    defer tx.Rollback()
    
    // Call service with transactional client
    entity, err := service.CreateEntity(ctx, tx.Client(), &input)
    if err != nil {
        return &model.Response{Success: false, Message: "Creation failed"}, nil
    }
    
    // Commit transaction
    if err := tx.Commit(); err != nil {
        return &model.Response{Success: false, Message: "Commit failed"}, nil
    }
    
    return &model.Response{Success: true, Entity: entity}, nil
}
```

**✅ Service method without transaction:**
```go
func (s *Service) CreateEntity(ctx context.Context, client *ent.Client, input *model.Input) (*ent.Entity, error) {
    // NO transaction here - use the client as provided
    entity, err := client.Entity.Create().
        SetName(input.Name).
        SetDescription(input.Description).
        Save(ctx)
    if err != nil {
        return nil, fmt.Errorf("creating entity: %w", err)
    }
    
    // Can do multiple operations with same client
    if input.AddRelations {
        err = s.addRelations(ctx, client, entity.ID, input.RelationIDs)
        if err != nil {
            return nil, fmt.Errorf("adding relations: %w", err)
        }
    }
    
    return entity, nil
}
```

**❌ NEVER do this in service layer:**
```go
func (s *Service) CreateEntity(ctx context.Context, client *ent.Client, input *model.Input) (*ent.Entity, error) {
    // ❌ WRONG - Creates nested transaction!
    tx, err := client.Tx(ctx)
    if err != nil {
        return nil, err
    }
    defer tx.Rollback()
    
    // This causes "error.chat.create_failed" and nested transaction errors
    // ...
}
```

#### Common Transaction Errors and Solutions:
- **"error.chat.create_failed"** with nested transaction → Remove transaction from service method
- **"transaction has already been committed"** → Use `Unwrap()` only in resolvers, not services
- **Privacy rules failing** → Ensure context has proper user information before creating transaction

### 4. Service Method Patterns
```go
// Correct pattern - explicit client parameter, NO transaction
func (s *Service) CreateEntity(ctx context.Context, client *ent.Client, input *model.CreateInput) (*ent.Entity, error) {
    // Business logic implementation
    // Use the provided client directly
}

// For operations that need to work with or without transaction
func (s *Service) UpdateEntity(ctx context.Context, client *ent.Client, id uuid.UUID, input *model.UpdateInput) (*ent.Entity, error) {
    // Works correctly whether client is transactional or not
    return client.Entity.UpdateOneID(id).
        SetName(input.Name).
        Save(ctx)
}
```

### 5. GraphQL Query Optimization with CollectFields

When implementing GraphQL resolvers, use the **combined approach** for optimal performance:

#### CollectFields Pattern
`CollectFields` automatically analyzes the GraphQL query context to determine which fields and relations are requested:

```go
// Automatically determines needed relations from GraphQL context
query, err := query.CollectFields(ctx)
if err != nil {
    return nil, err
}
```

#### Combined Approach (CollectFields + Explicit WithXXX)
For complex queries with computed fields, combine both approaches:

```go
func (r *queryResolver) Chats(ctx context.Context, ...) (*ent.ChatConnection, error) {
    query := r.client.Chat.Query()
    
    // 1. First, use CollectFields for automatic optimization
    query, err := query.CollectFields(ctx)
    if err != nil {
        return nil, err
    }
    
    // 2. Then, explicitly preload relations needed for computed fields
    query = query.WithMembers(func(mq *ent.ChatMemberQuery) {
        mq.WithUser().WithChat()
    })
    
    return query.Paginate(ctx, ...)
}
```

#### When to Use Each Approach:
1. **CollectFields alone**: For simple queries without computed fields
2. **Explicit WithXXX alone**: When you always need specific relations regardless of query
3. **Combined approach**: When you have:
    - Computed fields that require specific relations (e.g., `otherUser`, `unreadCount`)
    - Complex resolvers that access nested data
    - Need to prevent N+1 queries in field resolvers

#### Important Notes:
- CollectFields optimizes based on the GraphQL query, but may not detect all dependencies for computed fields
- Explicit WithXXX ensures relations are always loaded, preventing N+1 problems
- The combined approach provides the best of both worlds: automatic optimization + guaranteed data availability

### 6. UpdateOne Exec vs Save, Cache-first in Hooks/Services

- When updating entities and the returned object is not needed, prefer `Exec(ctx)` over `Save(ctx)` to avoid extra SELECTs after update.
  ```go
  // Prefer Exec when you don't need the updated entity
  err := client.Ticket.UpdateOne(t).SetSLAFirstResponseTime(now).Exec(ctx)
  ```
- In hooks/services that frequently read small reference entities (e.g., `TicketStatus`), use request-scoped cache first and only query DB on cache miss. After DB read, store the entity back to the cache using typed helpers (see `cache/keys.go`).
- For permission checks (privacy), memoize boolean results per-request (e.g., `perm:update:<ticketID>:<userID>:<role>`) to prevent duplicate EXISTS queries within the same request.

## Architecture Overview

### Directory Structure
- `/ent`: Entity framework schemas and generated code
- `/graph`: GraphQL API (schema, resolvers, directives)
- `/services`: Business logic layer by domain
- `/privacy`: Access control and permissions
- `/hooks`: Database hooks for audit logging
- `/middleware`: HTTP middleware components
- `/notifications`: Email/push notification system
- `/websocket`: Real-time subscription service
- `/locales`: Internationalization files
- `/tests/integration`: Comprehensive integration tests

### Key Systems

#### Query Logging System
- **Location**: `/querylog/` - система логирования GraphQL запросов
- **Log Storage**: `/query_logs/YYYY-MM-DD/HH-MM-SS/OperationName_SessionID.json`
- **Configuration**:
    - `ENABLE_QUERY_LOG=true` - включить логирование (только non-production)
- **Log Contents**:
    - GraphQL операция (имя, тип, raw query)
    - Все SQL запросы с аргументами и временем выполнения
    - Отладочные логи приложения (`debug_logs`) - все вызовы `utils.Logger` во время запроса
    - Время выполнения и метрики
- **Usage**: Анализ производительности, поиск N+1 проблем, отладка бизнес-логики
- **Important**: `.env` должен загружаться ДО `utils.InitLogger()` в main.go
- **Example log analysis**:
  ```bash
  # Найти медленные запросы (>100ms)
  grep -r '"duration_ms":[0-9]\{3,\}' query_logs/
  
  # Найти запросы с ошибками
  grep -r '"level":"ERROR"' query_logs/
  
  # Анализ конкретной операции
  find query_logs -name "UpdateTicket_*.json" -exec jq '.sql_queries | length' {} \;
  ```

#### Localization
- Source files: `/locales/*_en.json` and `/locales/*_ru.json`
- Auto-generated: `/locales/build/en.json` and `/locales/build/ru.json`
- Run `go generate` after adding new localization files
- Usage: `utils.T(ctx, "key.path")`

##### Checking Missing Localization Keys
After adding new translations in code:
1. Run the translation check tool:
   ```bash
   go run ./tools/check_translations/main.go
   ```
2. The tool will output any missing keys across all localization files
3. Add missing keys to appropriate `*_en.json` and `*_ru.json` files
4. Run `go generate` to rebuild the localization files
5. Verify no keys are missing by running the check tool again

**Important**: Always check for missing keys before committing to avoid runtime errors.

#### Notification System
- Event-driven with automatic detection
- Bulk notification threshold: 3+ recipients
- Template engine with variable substitution
- SLA notifications with exponential backoff

#### WebSocket Events
Standard event structure for all entities:
```go
type EntityEvent struct {
    Action   EntityAction  `json:"action"`     // created, updated, deleted
    EntityID uuid.UUID     `json:"entity_id"`
    Type     string        `json:"type"`       // ticket, user, notification, etc.
    Metadata map[string]any `json:"metadata,omitempty"`
}
```

#### Ticket Numbering
- Automatic generation based on ticket type
- Templates: Default (`000001`), Prefixed (`INC-000001`), Year-based (`2024-000001`)
- Type-based prefixes: incident→INC, request→REQ, problem→PRB, change→CHG

### Security Model
- Multi-tenant isolation with strict data boundaries
- Role-based access control (Admin, Manager, Agent, Client roles)
- Department-based team management
- Privacy rules enforced at database level using Ent privacy layer

## Testing Approach
- Unit tests colocated with source files (`*_test.go`)
- Integration tests in `/tests/integration`
- Use testcontainers for database testing
- Always verify with `make test` before committing

### Quick Testing Workflow
```bash
# Find failing tests quickly
make test-failures

# Run specific failing test
make test-single TEST="TestNotificationEventPrivacyRules"

# Fix issues and re-run
make test-single TEST="TestNotificationEventPrivacyRules"

# Run all integration tests after fixes
make test-integration
```

### Common Test Failure Patterns
- **Privacy/Auth errors**: `authentication required: ent/privacy: deny rule`
    - Solution: Use `privacy.WithSystemContext(ctx)` for seed data access
- **Field update errors**: `field X is not allowed for update`
    - Solution: Add field to allowed fields map using field constants
- **Import conflicts**: `privacy redeclared as imported package`
    - Solution: Use proper import aliases (`mainprivacy "main/privacy"`)
- **Build failures**: Missing dependencies or circular imports
    - Solution: Check imports and run `make generate` if needed

### System Context in Tests
Use `privacy.WithSystemContext(ctx)` when:
- Reading seed data (statuses, departments, etc.) in test setup
- Creating test entities that bypass privacy rules
- Accessing data outside user's normal permissions
```go
systemCtx := privacy.WithSystemContext(ctx)
status, err := client.TicketStatus.Query().Where(...).First(systemCtx)
```

### TestHelper Pattern для Интеграционных Тестов
**КРИТИЧЕСКИ ВАЖНО**: При написании интеграционных тестов используйте `TestHelper` pattern вместо Suite pattern с отдельными клиентами:

#### ❌ Неправильно (Suite pattern - вызывает "sql: database is closed")
```go
type MySuite struct {
    suite.Suite
    client *ent.Client  // ПРОБЛЕМА: отдельный клиент
    ctx    context.Context
}

func (s *MySuite) SetupSuite() {
    setup := GetGlobalTestSetup(s.T())
    s.client = setup.GetClient()  // Создает отдельное подключение
}

func (s *MySuite) TearDownSuite() {
    s.client.Close()  // ПРОБЛЕМА: закрывает глобальное подключение!
}
```

#### ✅ Правильно (TestHelper pattern - решает проблему)
```go
type MySuite struct {
suite.Suite
helper *TestHelper  // Используем TestHelper
ctx    context.Context
}

func (s *MySuite) SetupSuite() {
s.helper = NewTestHelper(s.T())  // Создает транзакционный контекст
s.ctx = s.helper.GetContext()
}

func (s *MySuite) TearDownSuite() {
s.helper.Rollback()  // Откатывает транзакцию, не закрывает подключение
}

func (s *MySuite) TestSomething() {
// Для обычных операций (включая системный контекст)
client := s.helper.GetClient()
systemCtx := privacy.WithSystemContext(s.ctx)

// Системный контекст работает с тем же транзакционным клиентом
data, err := client.Entity.Query().All(systemCtx)

// Для создания/обновления с системными правами
entity, err := client.Entity.Create().
SetName("test").
Save(systemCtx)
}
```

#### Почему TestHelper решает проблему:
1. **Suite pattern**: создавал отдельные клиенты и вызывал `client.Close()` → "sql: database is closed"
2. **TestHelper pattern**: использует транзакционный клиент → автоматическая изоляция без закрытия соединений
3. **Системный контекст**: прекрасно работает с транзакционным клиентом в TestHelper
4. **Rollback()**: автоматически очищает данные теста без влияния на другие тесты

### Constants over Hardcoded Strings
Always prefer constants over hardcoded strings:
- **Database field names**: Use generated Ent field constants (e.g., `notificationevent.FieldIsRead` instead of `"is_read"`)
- **Error messages**: Use constants from `privacy/errors.go`
- **Entity names**: Use generated type constants
- **Configuration values**: Define in constants or config files

Example:
```go
// ❌ Bad - hardcoded field names
allowedFields := map[string]bool{
    "is_read": true,
    "deleted_at": true,
}

// ✅ Good - using constants
allowedFields := map[string]bool{
    notificationevent.FieldIsRead: true,
    notificationevent.FieldDeletedAt: true,
}
```

### Code Generation Commands
When modifying Ent schemas or GraphQL, use these commands to regenerate code:

```bash
# Generate Ent code (schemas, queries, mutations)
go run -mod=mod ./ent/entc.go

# Generate GraphQL code (resolvers, schema)
go run -mod=mod github.com/99designs/gqlgen

# Or use Makefile (if updated)
make generate
```

**Important**: Always run both commands after modifying:
- Ent schemas in `ent/schema/`
- GraphQL schemas in `graph/`
- Privacy rules in `privacy/`

## Apollo Federation Setup

### Current Federation Implementation

This service supports Apollo Federation v2 with the following setup:

#### 1. Federation Configuration Files

- **`gqlgen.yml`**: Contains federation configuration
  ```yaml
  federation:
    filename: graph/generated/federation.go
    package: generated
    version: 2
  ```

- **`graph/schema/federation.graphql`**: Federation schema with directives
  ```graphql
  extend type User @key(fields: "id")
  extend type UserDepartment @key(fields: "id")
  ```

#### 2. Manual Implementation Required (entgql v0.6.0 limitations)

Due to current entgql limitations, the following must be implemented manually:

##### a. Entity Interface (`ent/gql_entity.go`)
```go
// Code generated manually for federation support. DO NOT DELETE.
func (*User) IsEntity() {}
func (*UserDepartment) IsEntity() {}
```

##### b. Entity Resolvers (`graph/resolvers/entity.resolvers.go`)
```go
func (r *entityResolver) FindUserByID(ctx context.Context, id uuid.UUID) (*ent.User, error) {
    // Uses DataLoader to prevent N+1 queries
    return dataloader.GetFederationUser(ctx, id)
}
```

##### c. Resolver Registration (`graph/resolvers/resolver.go`)
```go
func (r *Resolver) Entity() generated.EntityResolver { 
    return &entityResolver{r} 
}
```

#### 3. DataLoader Integration for Federation

Federation entity resolvers use DataLoaders to prevent N+1 queries:

- **`graph/dataloader/federation_user_loader.go`**: Batch loads users
- **`graph/dataloader/federation_department_loader.go`**: Batch loads departments

These are registered in `loaders.go` and accessible via:
- `dataloader.GetFederationUser(ctx, id)`
- `dataloader.GetFederationDepartment(ctx, id)`

### Adding New Federated Entities

To add a new entity to federation:

#### 1. Add @key directive in `federation.graphql`:
```graphql
extend type YourEntity @key(fields: "id")
```

#### 2. Add IsEntity() method in `ent/gql_entity.go`:
```go
func (*YourEntity) IsEntity() {}
```

#### 3. Create DataLoader in `graph/dataloader/`:
```go
type FederationYourEntityReader struct {
    client *ent.Client
}

func (r *FederationYourEntityReader) GetEntitiesByID(ctx context.Context, ids []uuid.UUID) ([]*ent.YourEntity, []error) {
    // Batch load implementation
}
```

#### 4. Add to Loaders struct and initialization:
```go
// In loaders.go
FederationYourEntityLoader *BatchLoader[uuid.UUID, *ent.YourEntity]

// In NewLoaders()
FederationYourEntityLoader: NewBatchLoader(reader.GetEntitiesByID, 2*time.Millisecond, 100),

// Add helper function
func GetFederationYourEntity(ctx context.Context, id uuid.UUID) (*ent.YourEntity, error) {
    return For(ctx).FederationYourEntityLoader.Load(ctx, id)
}
```

#### 5. Implement entity resolver in `entity.resolvers.go`:
```go
func (r *entityResolver) FindYourEntityByID(ctx context.Context, id uuid.UUID) (*ent.YourEntity, error) {
    return dataloader.GetFederationYourEntity(ctx, id)
}
```

#### 6. Regenerate code:
```bash
go generate ./...
```

### Important Notes

- **UUID Type**: All IDs use UUID type with custom marshaling in `ent/schema/uuidgql/uuidgql.go`
- **No Model Annotation Required**: Don't add User/UserDepartment to models in gqlgen.yml
- **DataLoader is Critical**: Always use DataLoader in entity resolvers to prevent N+1 queries
- **Manual Files**: Don't delete `gql_entity.go` and `entity.resolvers.go` - they're manually maintained

### Testing Federation

Export and check schema:
```bash
go run . -schema
grep "@key" schema.graphql
```

Deploy to Apollo Studio:
```bash
# With federation enabled
APOLLO_USE_FEDERATION=true go run . -schema
```

## Adding New Entities

When implementing a new entity in the system, follow this sequence:

### 1. Create Ent Schema (`/ent/schema/entity.go`)
```go
package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/edge"
	localmixin "main/ent/mixin"
	"main/privacy/entity"
)

type Entity struct {
	ent.Schema
}

func (Entity) Mixin() []ent.Mixin {
	return []ent.Mixin{
		localmixin.TimeMixin{},
		// localmixin.SoftDeleteMixin{}, // if soft delete needed
	}
}

func (Entity) Policy() ent.Policy {
	return ent.Policy{
		Query:    entity.QueryRule(),
		Mutation: entity.MutationRule(),
	}
}

func (Entity) Fields() []ent.Field {
	// Define fields with GraphQL annotations
}

func (Entity) Edges() []ent.Edge {
	// Define relationships
}
```

### 2. Create Privacy Rules (`/privacy/entity/`)
Create three files:
- `permissions.go` - Permission check functions
- `predicates.go` - Query predicates
- `privacy_rules.go` - Query/Mutation rules

### 3. Generate Code
```bash
make generate
```

### 4. Create GraphQL Schema (`/graph/schema/entity.graphql`)
```graphql
extend type Query {
    entities(filter: EntityFilter): [Entity!]! @member
    entity(id: UUID!): Entity @member
}

extend type Mutation {
    createEntity(input: CreateEntityInput!): EntityResponse! @admin
    updateEntity(id: UUID!, input: UpdateEntityInput!): EntityResponse! @admin
    deleteEntity(id: UUID!): DeleteResponse! @admin
}
```

### 5. Create Service Layer (`/services/entity/`)
```go
func (s *Service) CreateEntity(ctx context.Context, client *ent.Client, input *model.CreateEntityInput) (*ent.Entity, error) {
// Business logic with explicit client parameter
}
```

### 6. Implement Resolvers (`/graph/resolvers/entity.resolvers.go`)
- Auto-generated by gqlgen
- Add transaction management
- Use Unwrap() before returning entities

### 7. Write Integration Tests (`/tests/integration/`)
Test files:
- `entity_privacy_test.go` - Privacy rule tests
- `entity_crud_test.go` - CRUD operations
- `entity_edge_test.go` - Edge cases

### Key Points:
- **Always** pass client explicitly in service methods
- **Use** transactions for mutations
- **Add** post-commit hooks for external operations
- **Test** privacy rules comprehensively
- **Follow** existing patterns in the codebase

## Integration Test Patterns

### TestHelper vs Suite Pattern
Кратко: используйте `TestHelper` вместо отдельных клиентов БД (Suite pattern), чтобы избежать закрытия глобального подключения и проблем с транзакциями. Подробный пример уже приведён выше в разделе «TestHelper Pattern для Интеграционных Тестов».

### Transaction Abort Detection
При работе с транзакционными тестами добавляйте проверку состояния транзакции:

```go
func (suite *MySuite) TestSomething() {
    // Create user context first
    ctx := withUserContext(suite.ctx, testUser)
    ctx = ent.NewContext(ctx, suite.helper.GetClient())

    // Skip if transaction is aborted - test with user context
    _, err := suite.helper.GetClient().Entity.Query().Count(ctx)
    if err != nil && (err.Error() == "pq: current transaction is aborted, commands ignored until end of transaction block") {
        suite.T().Skip("Skipping test due to aborted transaction from previous test")
        return
    }
    
    // Основной код теста...
}
```

### Helper Function Protection
Кратко: проверяйте входные параметры в helper-функциях и логируйте/возвращайте ранний выход при некорректных значениях. Развёрнутый пример приведён ниже в разделе «Защита Helper Functions».

### SKIP vs FAIL Status
- **SKIP** - тест имеет защиту от абортированных транзакций и корректно пропускается
- **FAIL** - тест не имеет такой защиты и падает с ошибкой транзакции

**Проблема**: Suite тесты создавали отдельные клиенты БД и вызывали `client.Close()`, что закрывало глобальное подключение для других тестов.

**Решение**: Использование `TestHelper` pattern с транзакционной изоляцией вместо отдельных подключений к БД.

## Comprehensive Integration Testing Guide

### КРИТИЧЕСКИ ВАЖНО: TestHelper Pattern
Кратко: не используйте отдельные клиенты БД в Suite-тестах; для каждого подтеста создавайте новый `TestHelper` и делайте `Rollback()` в конце. Это предотвращает «sql: database is closed» и проблемы с транзакциями.

### Правильная Структура Тестовых Функций

**✅ Базовый шаблон для интеграционных тестов:**
```go
func testEntityQueryRules(t *testing.T, helper *TestHelper, ctx context.Context) {
client := helper.GetClient()
testData := setupEntityTestData(t, client, ctx)  // Setup с проверкой ошибок

testCases := []struct {
name          string
user          *ent.User
expectedCount int
description   string
}{
{
name:          "User can see their entities",
user:          testData.userA,
expectedCount: 2,
description:   "User should see entities they have access to",
},
}

for _, tc := range testCases {
t.Run(tc.name, func(t *testing.T) {
userCtx := ctxkeys.SetUserID(ctx, tc.user.ID)
userCtx = ctxkeys.SetLocalUser(userCtx, tc.user)
ctxWithClient := ent.NewContext(userCtx, client)

entities, err := client.Entity.Query().All(ctxWithClient)
require.NoError(t, err)
require.Equal(t, tc.expectedCount, len(entities), tc.description)
})
}
}
```

### Setup Functions: Обязательная Обработка Ошибок

**❌ НЕ игнорируйте ошибки в setup:**
```go
// ❌ ПЛОХО - игнорирование ошибок приводит к nil объектам
func setupBadTestData(t *testing.T, client *ent.Client, ctx context.Context) *testData {
    chat, _ := client.Chat.Create().Save(ctx)  // Игнорирование ошибки!
    member := createMember(t, client, ctx, chat, user, role)  // chat может быть nil
    return &testData{chat: chat, member: member}
}
```

**✅ ВСЕГДА проверяйте ошибки:**
```go
// ✅ ПРАВИЛЬНО - проверка всех ошибок
func setupGoodTestData(t *testing.T, client *ent.Client, ctx context.Context) *testData {
    systemCtx := mainprivacy.WithSystemContext(ctx)
    timestamp := time.Now().UnixNano()  // Наносекунды для уникальности!
    
    // Создание пользователей с проверкой ошибок
    userA := createOrGetUser(t, client, systemCtx, 
        fmt.Sprintf("test_userA_%d@test.com", timestamp), 
        "Test", "UserA", true, "member")
    require.NotNil(t, userA, "Failed to create userA")
    
    // Создание чата с проверкой ошибок
    chat, err := client.Chat.Create().
        SetType("group").
        SetName("Test Chat").
        SetCreatedBy(userA).
        Save(systemCtx)
    require.NoError(t, err, "Failed to create chat")
    require.NotNil(t, chat, "Chat is nil after creation")
    
    return &testData{chat: chat, userA: userA}
}
```

### Защита Helper Functions

**✅ Всегда проверяйте параметры в helper функциях:**
```go
func createChatMemberHelper(t *testing.T, client *ent.Client, ctx context.Context, 
    chat *ent.Chat, user *ent.User, role chatmember.Role) *ent.ChatMember {
    
    // Проверка всех параметров
    if chat == nil {
        t.Logf("Cannot create chat member: chat is nil")
        return nil
    }
    if user == nil {
        t.Logf("Cannot create chat member: user is nil")
        return nil
    }
    
    member, err := client.ChatMember.Create().
        SetChat(chat).
        SetUser(user).
        SetRole(role).
        Save(ctx)
    if err != nil {
        t.Logf("Error creating chat member: %v", err)
        return nil
    }
    return member
}
```

### Уникальность Тестовых Данных

**❌ НЕ используйте Unix секунды:**
```go
// ❌ ПЛОХО - может давать дубликаты
timestamp := time.Now().Unix()
email := fmt.Sprintf("user_%d@test.com", timestamp)
```

**✅ Используйте UnixNano для уникальности:**
```go
// ✅ ПРАВИЛЬНО - гарантирует уникальность
timestamp := time.Now().UnixNano()
email := fmt.Sprintf("user_%d@test.com", timestamp)
```

### Работа с Ent Schemas

**❌ НЕ угадывайте названия полей:**
```go
// ❌ ПЛОХО - неправильные названия методов
view := client.MessageView.Create().
    SetViewer(user).        // Неправильно!
    SetViewerID(user.ID).   // Неправильно!
    Save(ctx)
```

**✅ Изучите schema файлы:**
```go
// ✅ ПРАВИЛЬНО - смотрим в ent/schema/messageview.go
// edge.To("user", User.Type) означает SetUser/WithUser
view := client.MessageView.Create().
    SetUser(user).          // Правильно!
    SetUserID(user.ID).     // Правильно!
    Save(ctx)
```

### Транзакционные Ошибки: Диагностика

**🔍 Признаки проблем:**
- `pq: current transaction is aborted, commands ignored until end of transaction block`
- SKIP тесты вместо PASS
- "sql: database is closed"

**✅ Решения:**
1. **TestHelper на каждый подтест** (не один на весь тест)
2. **Проверка ошибок в setup** (никогда не игнорировать)
3. **Правильные field names** (изучать schema)
4. **UnixNano для уникальности** (избегать дубликатов)

### Шаблон Исправления Падающих Тестов

**1. Диагностика:**
```bash
go test -tags=integration ./tests/integration -run "TestEntity" -v
```

**2. Если есть SKIP или транзакционные ошибки:**
```go
// Было (ПРОБЛЕМА):
func TestEntityRules(t *testing.T) {
    helper := NewTestHelper(t)           // Один Helper на весь тест
    defer helper.Rollback()
    
    t.Run("Test1", func(t *testing.T) {
        testEntityTest1(t, helper, ctx)  // Если падает, ломает всю транзакцию
    })
    
    t.Run("Test2", func(t *testing.T) {
        testEntityTest2(t, helper, ctx)  // SKIP из-за абортированной транзакции
    })
}

// Стало (РЕШЕНИЕ):
func TestEntityRules(t *testing.T) {
    t.Run("Test1", func(t *testing.T) {
        helper := NewTestHelper(t)       // Отдельный Helper
        defer helper.Rollback()
        ctx, cancel := context.WithTimeout(helper.GetContext(), 30*time.Second)
        defer cancel()
        testEntityTest1(t, helper, ctx)
    })
    
    t.Run("Test2", func(t *testing.T) {
        helper := NewTestHelper(t)       // Отдельный Helper
        defer helper.Rollback()
        ctx, cancel := context.WithTimeout(helper.GetContext(), 30*time.Second)
        defer cancel()
        testEntityTest2(t, helper, ctx)
    })
}
```

### Privacy Rules Testing

**✅ Правильное тестирование privacy:**
```go
func testEntityPrivacyRules(t *testing.T, helper *TestHelper, ctx context.Context) {
    client := helper.GetClient()
    testData := setupEntityTestData(t, client, ctx)
    
    t.Run("User can see own entities", func(t *testing.T) {
        userCtx := ctxkeys.SetUserID(ctx, testData.userA.ID)
        userCtx = ctxkeys.SetLocalUser(userCtx, testData.userA)
        ctxWithClient := ent.NewContext(userCtx, client)  // Важно для privacy!
        
        entities, err := client.Entity.Query().All(ctxWithClient)
        require.NoError(t, err)
        require.Greater(t, len(entities), 0)
    })
    
    t.Run("Admin can see all entities", func(t *testing.T) {
        adminCtx := ctxkeys.SetUserID(ctx, testData.adminUser.ID)
        adminCtx = ctxkeys.SetLocalUser(adminCtx, testData.adminUser)
        ctxWithClient := ent.NewContext(adminCtx, client)
        
        entities, err := client.Entity.Query().All(ctxWithClient)
        require.NoError(t, err)
        // Тестируем специфичную логику для админа
    })
}
```

### Быстрая Отладка Тестов

**1. Запуск конкретного теста:**
```bash
go test -tags=integration ./tests/integration -run "TestChatPrivacy" -v
```

**2. Запуск с подробным выводом:**
```bash
go test -tags=integration ./tests/integration -run "TestEntity" -v -count=1
```

**3. Поиск падающих тестов:**
```bash
make test-failures  # Быстрый анализ
```

## DataLoader Pattern for GraphQL N+1 Query Optimization

### Problem
When implementing computed fields in GraphQL (fields that require additional database queries), each item in a list causes a separate query, leading to N+1 performance issues.

### Solution: DataLoader Pattern
DataLoader consolidates multiple queries into batch operations, dramatically reducing database load.

### Implementation Steps

#### 1. Create DataLoader Reader (`/graph/dataloader/entity_loader.go`)
```go
package dataloader

import (
    "context"
    "main/ent"
    "github.com/google/uuid"
)

type EntityReader struct {
    client *ent.Client
}

func NewEntityReader(client *ent.Client) *EntityReader {
    return &EntityReader{client: client}
}

// GetEntities fetches multiple entities in a single query
func (r *EntityReader) GetEntities(ctx context.Context, ids []uuid.UUID) ([]*ent.Entity, []error) {
    results := make([]*ent.Entity, len(ids))
    errors := make([]error, len(ids))
    
    // Batch query all entities
    entities, err := r.client.Entity.Query().
        Where(entity.IDIn(ids...)).
        All(ctx)
    
    if err != nil {
        for i := range errors {
            errors[i] = err
        }
        return results, errors
    }
    
    // Map results back to original order
    entityMap := make(map[uuid.UUID]*ent.Entity)
    for _, e := range entities {
        entityMap[e.ID] = e
    }
    
    for i, id := range ids {
        if entity, ok := entityMap[id]; ok {
            results[i] = entity
        }
    }
    
    return results, errors
}
```

#### 2. Register DataLoader in Loaders (`/graph/dataloader/loaders.go`)
```go
type Loaders struct {
    // ... existing loaders
    EntityLoader *dataloadgen.Loader[uuid.UUID, *ent.Entity]
}

func NewLoaders(client *ent.Client) *Loaders {
    entityReader := NewEntityReader(client)
    
    return &Loaders{
        // ... existing loaders
        EntityLoader: dataloadgen.NewLoader(
            entityReader.GetEntities,
            dataloadgen.WithWait(2*time.Millisecond),
        ),
    }
}

// Helper functions
func GetEntity(ctx context.Context, id uuid.UUID) (*ent.Entity, error) {
    loaders := For(ctx)
    return loaders.EntityLoader.Load(ctx, id)
}
```

#### 3. Update GraphQL Resolver
```go
// Before (N+1 problem):
func (r *parentResolver) Entity(ctx context.Context, obj *ent.Parent) (*ent.Entity, error) {
    return r.client.Entity.Query().
        Where(entity.HasParentWith(parent.ID(obj.ID))).
        Only(ctx)
}

// After (with DataLoader):
func (r *parentResolver) Entity(ctx context.Context, obj *ent.Parent) (*ent.Entity, error) {
    return dataloader.GetEntity(ctx, obj.EntityID)
}
```

#### 4. DataLoader Middleware is Applied Automatically
The DataLoader middleware is already configured in `/server/server.go`:
```go
// Apply DataLoader middleware using the dataloader package directly
handler := dataloader.Middleware(client, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
graphqlServer := NewGraphQLServer(client, bundle)
graphqlServer.ServeHTTP(w, r)
}))
```

### Common Use Cases

#### 1. Last Message in Chats
```go
// Fetches last messages for multiple chats in one query
func (r *MessageReader) GetLastMessages(ctx context.Context, chatIDs []uuid.UUID) ([]*ent.Message, []error) {
// Implementation that gets all messages and filters for latest per chat
}
```

#### 2. Unread Count
```go
// Counts unread messages for multiple chats in batch
func (r *UnreadCountReader) GetUnreadCounts(ctx context.Context, chatIDs []uuid.UUID) ([]int, []error) {
    // Implementation that counts unread messages per chat
}
```

#### 3. Related Users
```go
// Fetches related users for multiple entities
func (r *OtherUserReader) GetOtherUsers(ctx context.Context, chatIDs []uuid.UUID) ([]*ent.User, []error) {
    // Implementation that finds other users in personal chats
}
```

### Benefits
- **Performance**: Reduces N queries to 1-3 queries
- **Automatic Batching**: Requests within 2ms window are batched
- **Caching**: Results cached within single GraphQL request
- **Transparent**: No changes needed in GraphQL schema

### When to Use DataLoader
- Computed fields that require additional queries
- Fields accessed in list contexts (edges, connections)
- Cross-entity lookups (e.g., latest related record)
- Aggregations (counts, sums, etc.)

### Important Notes
- DataLoader caches are request-scoped (not global)
- Always return results in same order as input IDs
- Handle nil/missing results gracefully
- Consider pagination limits for batch queries

### Advanced Optimization: Preload Cache

To prevent fragmented queries when DataLoaders need data that was already loaded in the main query, we use a PreloadCache system:

#### 1. PreloadCache Structure (`/graph/dataloader/preload_cache.go`)
```go
type PreloadCache struct {
    ChatMembers map[uuid.UUID][]*ent.ChatMember // chat ID -> members
    Users       map[uuid.UUID]*ent.User          // user ID -> user
}
```

#### 2. Populate Cache in Main Query
```go
func (r *queryResolver) Chats(ctx context.Context, ...) (*ent.ChatConnection, error) {
    // ... execute query with preloaded data
    
    // Populate cache with loaded data
    if connection != nil && len(connection.Edges) > 0 {
        cache := dataloader.GetPreloadCache(ctx)
        chats := make([]*ent.Chat, len(connection.Edges))
        for i, edge := range connection.Edges {
            chats[i] = edge.Node
        }
        cache.PopulateFromChats(chats)
    }
    
    return connection, nil
}
```

#### 3. Use Cache in DataLoaders
```go
func (r *OtherUserReader) GetOtherUsers(ctx context.Context, chatIDs []uuid.UUID) ([]*ent.User, []error) {
    // Check preload cache first
    cache := GetPreloadCache(ctx)
    
    // Only query for data not in cache
    for i, chatID := range chatIDs {
        if cachedMembers, ok := cache.ChatMembers[chatID]; ok {
            // Use cached data
            for _, member := range cachedMembers {
                if member.Edges.User != nil && member.Edges.User.ID != currentUserID {
                    results[i] = member.Edges.User
                    break
                }
            }
        } else {
            // Add to list for batch query
            needsLoading = append(needsLoading, chatID)
        }
    }
    
    // Query only missing data
    if len(needsLoading) > 0 {
        // Batch query for missing data
    }
}
```

#### 4. Minimize Data Transfer
When loading data for counting or simple checks, select only necessary fields:
```go
// Bad - loads full message content
messages, err := r.client.Message.Query().
Where(message.HasChatWith(chat.IDIn(chatIDs...))).
All(ctx)

// Good - loads only IDs for counting
messages, err := r.client.Message.Query().
Where(message.HasChatWith(chat.IDIn(chatIDs...))).
Select(message.FieldID). // Only select ID
WithChat(func(q *ent.ChatQuery) {
q.Select(chat.FieldID) // Only need chat ID
}).
All(ctx)
```

This approach reduces SQL queries from ~100+ to ~12 for complex GraphQL operations.

### CRITICAL: DataLoader Caching and WebSockets

**⚠️ ВАЖНО**: DataLoader'ы с кешированием несовместимы с WebSocket подписками!

#### Проблема
- DataLoader (например, `github.com/vikstrous/dataloadgen`) кеширует данные в контексте запроса
- WebSocket подписки используют долгоживущее соединение с одним контекстом
- Это приводит к возврату устаревших данных при последующих событиях подписки

#### Симптомы
- Первое событие WebSocket возвращает актуальные данные
- Последующие события возвращают закешированные (устаревшие) данные
- После переподключения WebSocket снова работает корректно (на один запрос)

#### Решение
Вместо DataLoader'ов с кешированием используйте BatchLoader без кеширования:
```go
// ❌ НЕ используйте DataLoader с кешированием для WebSocket
dataloadgen.NewLoader(fetch, dataloadgen.WithWait(2*time.Millisecond))

// ✅ Используйте BatchLoader без кеширования
NewBatchLoader(fetch, 2*time.Millisecond, 100)
```

#### Реализация BatchLoader
См. `/graph/dataloader/batchloader.go` - простая реализация, которая:
- Группирует запросы для оптимизации (batch loading)
- НЕ кеширует результаты между вызовами
- Подходит для WebSocket подписок и real-time данных

### Token Validation Caching

To reduce database queries for token validation, we implement Redis-based caching:

#### 1. Token Cache Implementation (`/services/auth/token_cache.go`)
- **Redis cache**: 1-minute TTL cache for reducing database queries across requests
- **Key pattern**: `tenant:{tenantID}/token_cache:{sessionToken}` (follows the same pattern as online status)
- **Request-scoped cache**: Use context values for caching within a single request

#### 2. Benefits
- Reduces token validation queries from 3-4 per validation to 0 (when cached)
- Skips session UPDATE queries for cached tokens (session updated at most once per minute)
- Maintains security by checking user active status even for cached tokens

#### 3. Usage
```go
// In middleware - use cached validation
userInfo, err := authService.ValidateTokenWithCache(ctx, client, token)

// On logout - invalidate cache
cache := GetTokenCache()
cache.InvalidateCachedToken(ctx, sessionToken)
```

#### 4. CRITICAL: Multi-tenant Caching Restrictions
**⚠️ ВАЖНО**: В мультитенантном мультиконтейнерном приложении ЗАПРЕЩЕНО использовать постоянный in-memory кэш!

- **НИКОГДА не используйте**: `sync.Map`, глобальные переменные или любой другой in-memory кэш между запросами
- **РАЗРЕШЕНО**:
    - Redis с изоляцией по tenant ID
    - Context values для кэширования в рамках одного запроса
    - Request-scoped переменные

**Причины**:
1. Контейнеры могут обслуживать разных тенантов
2. In-memory данные могут утечь между тенантами
3. Горизонтальное масштабирование требует stateless контейнеров

This optimization is especially effective for operations that trigger multiple token validations within a short time.

## Important Notes
- Never commit secrets or API keys
- Always use prepared statements for database queries
- Follow existing code patterns and conventions
- Use structured logging with Zap
- Maintain backward compatibility for API changes

Use `tree` to display folder structure in a tree-like format in the terminal