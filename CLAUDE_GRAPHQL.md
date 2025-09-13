## Паттерны GraphQL‑резолверов и производительность

Назначение: единые паттерны для Ent + gqlgen, чтобы исключать N+1, соблюдать правила хуков/транзакций и минимизировать количество запросов к БД.

### Основные принципы (Ent + правила проекта)
- Всегда передавайте `*ent.Client` явно в сервисы; не используйте `ent.FromContext` вне хуков.
- Для любой операции, которая запускает хуки/политики Ent, оборачивайте контекст: `ctxWithClient := ent.NewContext(ctx, client)` или внутри tx `txCtx := ent.NewTxContext(ctx, tx)`.
- Транзакции живут только в резолверах/сервисах. Побочные эффекты (события, pub/sub) только в `tx.OnCommit(...)`.
- Вызывайте `.Unwrap()` на сущностях, возвращаемых из tx, только в резолверах, непосредственно перед возвратом в GraphQL (никогда в сервисах/хуках).

### Загрузка по выборке: CollectFields + предзагрузка верхнего уровня
Проблема: gqlgen генерирует «ленивые» резолверы связей. Если связь не предзагружена, для каждого узла делается отдельный запрос (N+1).

Откуда берётся N+1 (ленивая связь):
```1203:1208:ent/gql_edge.go
func (tcf *TicketCommentFile) File(ctx context.Context) (*File, error) {
    result, err := tcf.Edges.FileOrErr()
    if IsNotLoaded(err) {
        result, err = tcf.QueryFile().Only(ctx)
    }
    return result, err
}
```

Правила:
- Для запросов/списков: вызывать `CollectFields(ctx)` у корневого билдера, чтобы подгрузить ровно запрошенные связи.
- Для мутаций: после коммита перечитывать сущность свежим запросом с `CollectFields(ctx)`.
- Предзагружать только верхний уровень списков/коллекций (например, `WithCommentFiles()`), а вложенные поля списков (например, `file`) отдавать через DataLoader (ниже).

Пример (верхний уровень):
```go
q := client.TicketComment.Query().Where(ticketcomment.ID(commentID))
q, _ = q.CollectFields(ctx)
// Предзагрузка списка связей (один запрос на список)
q = q.WithCommentFiles()
comment, err := q.Only(ctx)
if err != nil {
    // Fallback if reload failed
    comment = comment.Unwrap()
}
```

Чеклист после мутаций (перечитывание сущности):
- Делать `CollectFields(ctx)` у запроса сущности.
- Делать `With<Child>()` для предзагрузки списков верхнего уровня (если они нужны сразу).
- Вложенные поля коллекций отдавать через DataLoader — отдельная предзагрузка не нужна.
- Если перечитать не удалось — использовать `.Unwrap()`.

### DataLoader для ребёр (рекомендуемый паттерн для вложенных полей)
Для вложенных полей, которые часто становятся источником N+1 (например, `TicketCommentFile.file`), переводим поле на форс‑резолвер и грузим через DataLoader. Это устраняет N+1 независимо от alias и порядка предзагрузок.

Паттерн:
1) В `gqlgen.yml` включить принудительный резолвер для поля:
```yaml
models:
  TicketCommentFile:
    fields:
      file:
        resolver: true
```
2) Реализовать batch‑ридер (одним запросом), который возвращает значения в исходном порядке.
3) Резолвер поля:
    - Если edge уже предзагружен — вернуть его.
    - Иначе — вернуть из DataLoader: `return dataloader.GetXYZ(ctx, id)`.

Примеры в проекте:
- Permissions — `graph/dataloader/*` (`File.canDelete`, `Ticket.canDelete`).
- Вложенное поле `TicketCommentFile.file` переведено на DataLoader.

Важно:
- После перевода вложенного поля на DataLoader дополнительные Named‑варианты не используются.
- Предзагрузка родительских списков (`WithCommentFiles()`) может оставаться ради одного запроса на список, а вложенные поля списка грузятся батчем DataLoader‑ом.

### Списки и пагинация
- Ограничивать размер страницы.
- Вызывать `CollectFields(ctx)` у запроса списка.
- Применять фильтры/сортировку через сгенерированные хелперы.

Пример:
```go
query, err := client.TicketComment.Query().Where(ticketcomment.HasTicketWith(ticket.ID(ticketID))).CollectFields(ctx)
if err != nil { return nil, err }
return query.Paginate(ctx, after, first, before, last,
    ent.WithTicketCommentOrder(orderBy),
    ent.WithTicketCommentFilter(where.Filter),
)
```

### Кеш в рамках запроса
- HTTP GraphQL: request‑кеш создаётся автоматически middleware; вручную добавлять в резолверах/сервисах не требуется.
- Вне GraphQL (cron/worker/tests): оборачивайте контекст per‑operation:
```go
ctx = cache.WithRequestCache(ctx)
// при необходимости совместимости: ctx = ctxkeys.WithRequestCache(ctx)
```
- Чтение/запись через типизированные хелперы:
```go
if uploaderID, ok := cache.GetFromCache[uuid.UUID](ctx, cache.FileUploaderKey(fileID)); ok {
    // fast-path without DB
}

### Apollo Federation и Entity Resolvers

#### Проблема
При использовании Apollo Federation, gateway часто запрашивает множество сущностей по ID одновременно (entity resolution). Без оптимизации это приводит к N отдельным запросам к БД.

#### Решение: DataLoader для Entity Resolvers
Все entity resolvers должны использовать специальные federation DataLoader'ы для batch-загрузки:

```go
// graph/resolvers/entity.resolvers.go
func (r *entityResolver) FindUserByID(ctx context.Context, id uuid.UUID) (*ent.User, error) {
    // Использует FederationUserLoader для batch-загрузки
    return dataloader.GetFederationUser(ctx, id)
}
```

#### Реализация Federation DataLoader
```go
// graph/dataloader/federation_user_loader.go
func (r *FederationUserReader) GetUsersByID(ctx context.Context, userIDs []uuid.UUID) ([]*ent.User, []error) {
    // Batch-загрузка всех пользователей одним запросом
    users, err := r.client.User.Query().
        Where(user.IDIn(userIDs...)).
        All(ctx)
    
    // Возвращаем в том же порядке, что и запрошенные ID
    userMap := make(map[uuid.UUID]*ent.User)
    for _, u := range users {
        userMap[u.ID] = u
    }
    
    results := make([]*ent.User, len(userIDs))
    for i, id := range userIDs {
        results[i] = userMap[id]
    }
    
    return results, errors
}
```

#### Важные моменты
- **Отдельные DataLoader'ы для federation**: Не используйте обычные DataLoader'ы для entity resolution, т.к. они могут иметь другую логику предзагрузки
- **BatchLoader без кеширования**: Используйте `BatchLoader` вместо кеширующих DataLoader'ов для совместимости с WebSocket
- **Порядок результатов**: Всегда возвращайте результаты в том же порядке, что и входные ID

#### Добавление новой federated entity
1. Добавьте `@key` директиву в `graph/schema/federation.graphql`
2. Создайте метод `IsEntity()` в `ent/gql_entity.go`
3. Реализуйте DataLoader в `graph/dataloader/federation_<entity>_loader.go`
4. Добавьте loader в `graph/dataloader/loaders.go`
5. Реализуйте entity resolver с использованием DataLoader
```

// After loading from DB:
cache.SetInCache[uuid.UUID](ctx, cache.FileUploaderKey(fileID), uploaderID)
```
- Использовать фабрики ключей из `cache/keys.go` для всех строковых ключей. Не хардкодить строки.
- Инвалидировать каноничные представления перед мутациями (например, ключ деталей тикета), чтобы избежать «грязного» чтения в рамках запроса.

Подробнее: `cache/README.md`.

### UpdateOne: Exec вместо Save, когда не нужна сущность

Когда результат обновления (полная сущность) не используется, предпочтительно вызывать `Exec(ctx)` вместо `Save(ctx)`. Это предотвращает дополнительные SELECT, которые Ent выполняет после `Save` в некоторых конфигурациях.

Пример:
```go
// Было — возвращает сущность и может вызывать дополнительный SELECT
_, err := client.Ticket.UpdateOne(t).SetSLAFirstResponseTime(now).Save(ctx)

// Стало — без лишнего чтения
err := client.Ticket.UpdateOne(t).SetSLAFirstResponseTime(now).Exec(ctx)
```

### Cache-first для малых справочников в хуках/сервисах

Для частых справочных чтений в хуках (например, статусы тикетов), сначала проверяйте request‑cache и только при промахе читайте из БД, затем кладите в кэш:

```go
if s, ok := cache.GetFromCache[*ent.TicketStatus](ctx, cache.StatusKey(id)); ok && s != nil {
    return s
}
s, err := client.TicketStatus.Get(ctx, id)
if err != nil { return nil, err }
cache.SetInCache[*ent.TicketStatus](ctx, cache.StatusKey(s.ID), s)
```

### Транзакции и хуки — безопасный каркас
```go
tx, err := client.Tx(ctx)
if err != nil { /* return error */ }
defer func() { if err != nil { _ = tx.Rollback() } }()

txCtx := ent.NewTxContext(ctx, tx)
// business logic using tx.Client(), ent.NewContext on operations that trigger hooks

tx.OnCommit(func(ctx context.Context, _ *ent.Tx) error {
    // publish events, notify, etc.
    return nil
})

if err = tx.Commit(); err != nil { /* return commit error */ }

// Reload entity for GraphQL using selection-aware loading (see above)
```

### Рекомендуется / Не рекомендуется
Рекомендуется:
- Делать `CollectFields` для запросов и финальной догрузки после мутаций.
- Предзагружать списки верхнего уровня (например, `WithCommentFiles()`), если они нужны сразу.
- Переводить вложенные поля коллекций на DataLoader.
- Использовать кеш запроса для повторных обращений в рамках одного запроса.
- Побочные эффекты — только в `OnCommit`; `.Unwrap()` — только в резолверах.

Не рекомендуется:
- Полагаться на «ленивые» связи — это приводит к N+1.
- Вызывать `.Unwrap()` в сервисах/хуках.
- Использовать `ent.FromContext` вне хуков.
- Хардкодить строковые ключи кеша (только фабрики из `cache/keys.go`).

### Пример: паттерн резолвера мутации (компактный)
```go
func (r *mutationResolver) CreateTicketComment(ctx context.Context, ticketID uuid.UUID, input model.CreateTicketCommentInput) (*model.TicketCommentResponse, error) {
    tx, err := r.client.Tx(ctx)
    if err != nil { /* return */ }
    defer func() { if err != nil { _ = tx.Rollback() } }()

    txCtx := ent.NewTxContext(ctx, tx)
    svc := comment.GetCommentService()
    newComment, err := svc.CreateComment(txCtx, tx.Client(), ticketID, input)
    if err != nil { /* return */ }

    newID := newComment.ID
    tx.OnCommit(func(ctx context.Context, _ *ent.Tx) error { /* publish events */; return nil })
    if err = tx.Commit(); err != nil { /* return */ }

    // Selection-aware reload: предзагружаем список, а вложенные поля отдаются через DataLoader
    q := r.client.TicketComment.Query().Where(ticketcomment.ID(newID))
    q, _ = q.CollectFields(ctx)
    q = q.WithCommentFiles()
    newComment, err = q.Only(ctx)
    if err != nil { newComment = newComment.Unwrap() }

    return &model.TicketCommentResponse{ Success: true, Message: utils.T(ctx, "success.comment.created"), Comment: newComment }, nil
}
```

### Когда добавлять DataLoader
Добавляйте DataLoader, если:
- Поле часто запрашивается в разных контекстах и предзагрузка неудобна/избыточна.
- Требуются батч‑проверки прав/расчёты.

Шаги внедрения:
- Добавить batch‑ридер и API в `graph/dataloader/*`.
- Зарегистрировать лоадеры в контексте запроса (middleware).
- В `gqlgen.yml` включить `resolver: true` для поля и реализовать резолвер с использованием лоадера.

---
Следуя этим паттернам, вы избегаете N+1 (и на верхнем уровне, и во вложенных полях), сохраняете корректную семантику транзакций/хуков и стабилизируете задержки при сложных выборках.


### Хуки и сервисы: предотвращение N+1 (каноническая загрузка тикета + request‑cache)

Проблема: даже при оптимальных резолверах N+1 может возникать в сервисном слое и хуках (audit, SLA tracking), если там используются «ленивые» загрузки (loadX) и повторные запросы тикета/edges.

Решение (протокол):
- В сервисе мутации (до Save/Exec) один раз загрузить «канонический» тикет со всеми нужными связями и положить в кэш под унифицированным ключом:
```go
full, _ := repo.GetTicketWithDetails(ctx, ticketID) // WithSubscriptions(WithUser), WithStatus, WithPriority, WithType, WithDepartment
cache.SetInCache(ctx, cache.TicketDetailsKey(ticketID), full)
```
- В хуках (status tracking, audit) перед любыми запросами сначала пытаться брать тикет из кэша:
```go
if t, ok := cache.GetFromCache[*ent.Ticket](ctx, cache.TicketDetailsKey(ticketID)); ok && t != nil {
    // использовать t и его edges, не делать loadX и повторные запросы
} else {
    // fallback: единожды загрузить полный тикет и положить в кэш
}
```
- Избегать ленивых загрузок в хуках (loadDepartment/loadSubscriptions и т.п.). Только предзагруженные edges из канонического тикета или данные из кэша.
- Инвалидация канонического представления перед мутациями и переустановка после коммита при необходимости:
```go
cache.DeleteFromCache[*ent.Ticket](ctx, cache.TicketDetailsKey(ticketID))
// После успешного коммита можно перечитать и снова положить в кэш
```

Эффект: повторные запросы к `tickets`, `ticket_statuses`, `ticket_types`, `ticket_priorities`, `departments`, `ticket_subscriptions`, `users` исчезают; хуки используют одно каноническое представление тикета в рамках запроса.



### Батч‑логирование мутаций через хуки Ent (устранение N+1 при массовых вставках)

Проблема: при массовом создании записей (например, подписок) каждое срабатывание хука пишет отдельную строку аудита и может выполнять дополнительные SELECT, что ведёт к N+1. Нужно сохранить семантику «одна запись аудита на одну созданную сущность», но выполнять запись аудита одним батчем.

Системный подход (расширяемый на любые сущности):

- Контекстный флаг «batch logging»
    - Ввести вспомогательную функцию (примерная сигнатура): `ctx = ticketaudit.WithBatchLogging(ctx)`.
    - Флаг читается в хуках: если включён — хук не вызывает `Save` лога сразу, а отправляет событие в накопитель (collector).

- Накопитель событий (request‑scoped collector)
    - Реализовать типизированный collector поверх request‑cache:
        - `ticketaudit.AddCreate(ctx, payload)` — кладёт payload в коллекцию для текущего запроса.
        - `ticketaudit.DrainCreates(ctx) []CreatePayload` — возвращает и очищает коллекцию.
    - Payload содержит только необходимые поля для батч‑вставки (без повторных загрузок): `Action`, `TicketID`, `EntityID` (например, `SubscriptionID`), опц. `UserID`, `ActionDescription`, `NewValues`/`OldValues`, `RequestID`, `Metadata`.

- Изменения в хуках (пример на create)
    - В хукe create:
        - Не перечитывать сущность (использовать результат мутации).
        - Если `batch logging` активен — сформировать payload и положить в collector; вернуть `next.Mutate` результат без сохранения аудита.
        - Если `batch logging` выключен — сохранить лог как раньше (обратная совместимость для одиночных операций).

- Сохранение аудита после `CreateBulk`
    - В сервисе/резолвере, где вызывается `CreateBulk(...)`:
        - Перед вызовом включить `WithBatchLogging(ctx)`.
        - После успешного `CreateBulk` вызвать `payloads := ticketaudit.DrainCreates(ctx)` и выполнить один `TicketSubscriptionAuditLog.CreateBulk` c генерацией билдеров из payload’ов.
        - Для обхода privacy-директив использовать `privacy.WithSystemContext(ctx)` при сохранении логов (как в существующих сервисах аудита).
        - При необходимости зарегистрировать сохранение в `tx.OnCommit(...)`, если сам `CreateBulk` выполнялся внутри транзакции.

- Минимизация запросов в батч‑режиме
    - Не загружать сущность повторно в хукe (убрать дополнительный SELECT).
    - В payload класть ID сущностей; для `CreateBulk` логов достаточно `SetTicketID(...)`, `SetSubscriptionID(...)`, `SetUserID(...)` и т.п., без предзагрузок.

- Пример (схема действий, псевдокод)

```go
// В сервисе массового создания
ctx = cache.WithRequestCache(ctx)
ctx = ticketaudit.WithBatchLogging(ctx)
ctxWithClient := ent.NewContext(ctx, client)

subs, err := client.TicketSubscription.CreateBulk(builders...).Save(ctxWithClient)
if err != nil { /* handle */ }

// Сохраняем логи одним батчем
payloads := ticketaudit.DrainCreates(ctx)
if len(payloads) > 0 {
    sys := privacy.WithSystemContext(ctx)
    sys = ent.NewContext(sys, client)
    auditBuilders := make([]*ent.TicketSubscriptionAuditLogCreate, 0, len(payloads))
    for _, p := range payloads {
        b := client.TicketSubscriptionAuditLog.Create().
            SetAction(ticketsubscriptionauditlog.ActionCreate).
            SetActionDescription(p.ActionDescription).
            SetTicketID(p.TicketID).
            SetSubscriptionID(p.EntityID).
            SetNewValues(p.NewValues)
        if p.UserID != uuid.Nil { b.SetUserID(p.UserID) }
        if p.RequestID != "" { b.SetRequestID(p.RequestID) }
        if len(p.Metadata) > 0 { b.SetMetadata(p.Metadata) }
        auditBuilders = append(auditBuilders, b)
    }
    _, _ = client.TicketSubscriptionAuditLog.CreateBulk(auditBuilders...).Save(sys)
}
```

- Расширяемость
    - Шаблон одинаков для других сущностей (комментарии, файлы, оценки, прочтения):
        - Контекстный флаг `WithBatchLogging` (общий пакет аудита или по доменам).
        - Collector на request‑cache для каждого типа события (`Create/Update/Delete`).
        - Модификация соответствующих хуков (`With...AuditLog`) с режимом enqueue‑vs‑save.
        - Единая точка `CreateBulk` в сервисе, где выполняется `Drain + CreateBulk` логов.

Результат: одна вставка в основную таблицу (`CreateBulk`) и одна вставка логов (`CreateBulk`), без N+1 и без изменения семантики «одна запись аудита на одну сущность».

