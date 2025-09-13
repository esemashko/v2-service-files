# Subscription Service

Сервис для унифицированной работы с подписками на события в системе через Redis Pub/Sub.

## Назначение

Сервис инкапсулирует общую логику подписок для различных типов сущностей:
- Проверку наличия tenant в контексте
- Получение Redis-клиента
- Формирование имени канала на основе типа сущности и идентификатора
- Обработку сообщений и ошибок
- Корректное закрытие подписки при завершении контекста

## Модель событий

Сервис использует универсальную структуру `EntityEvent` для всех типов событий:

```go
type EntityAction string

// Константы для действий с сущностями
const (
    EntityActionCreated EntityAction = "created"
    EntityActionUpdated EntityAction = "updated"
    EntityActionDeleted EntityAction = "deleted"
)

type EntityEvent struct {
    Action   EntityAction  `json:"action"`
    EntityID uuid.UUID     `json:"entity_id"`
    Type     string        `json:"type"`
    Metadata map[string]any `json:"metadata,omitempty"`
}
```

Где:
- `Action` - типизированное действие: EntityActionCreated, EntityActionUpdated, EntityActionDeleted
- `EntityID` - ID сущности, к которой относится событие
- `Type` - тип сущности: "ticket", "user", "notification", etc.
- `Metadata` - дополнительные данные о событии (опционально)

## Реализованные подписки

### 1. Подписка на обновления тикетов
- **Канал (по ID)**: `{tenantID}:ticket_{ticketID}`
- **Канал (глобальный список)**: `{tenantID}:ticket:updates`
- **Тип события**: `ticket`
- **Использование**: Отслеживание изменений в конкретном тикете

### 2. Подписка на комментарии тикетов
- **Канал**: `{tenantID}:ticket_comment:updates`
- **Тип события**: `ticket_comment`
- **Использование**: Отслеживание новых комментариев к тикетам
### 4. Подписка на список тикетов (глобальный канал)
- **Канал**: `{tenantID}:ticket:updates`
- **Тип события**: `ticket`
- **Использование**: Для обновления грида списка тикетов при создании/обновлении/удалении

Ответ GraphQL: `TicketListItemResponse` с полями:
- `success`, `message`
- `action`: CREATED | UPDATED | DELETED
- `ticket`: Ticket (nil для DELETED)
- `id`: ID тикета (всегда заполняется)


### 3. Подписка на уведомления пользователя
- **Канал**: `{tenantID}:notification_user_{userID}`
- **Тип события**: `notification`
- **Использование**: Отслеживание новых уведомлений для конкретного пользователя
- **Особенность**: Подписка ограничена уведомлениями конкретного пользователя

## Использование

### Пример подписки на уведомления пользователя

```go
// Резолвер подписки на уведомления (уже реализован)
func (r *subscriptionResolver) UpdatedNotification(ctx context.Context) (<-chan *model.NotificationResponse, error) {
    userID := ctxkeys.GetUserID(ctx)
    ch := make(chan *model.NotificationResponse, 1)
    
    subscriptionService := subscription.New()
    userIDStr := userID.String()
    channel, err := subscriptionService.BuildChannelName(ctx, "notification_user", &userIDStr)
    if err != nil {
        close(ch)
        return nil, err
    }
    
    eventHandler := func(ctx context.Context, payload []byte) error {
        var evt subscription.EntityEvent
        if err := json.Unmarshal(payload, &evt); err != nil {
            return err
        }
        
        if evt.Type != "notification" {
            return nil
        }
        
        // Получаем актуальные данные уведомления
        notificationEvent, err := r.client.NotificationEvent.Query().
            Where(notificationevent.ID(evt.EntityID)).
            WithTicket().
            Only(ctx)
        if err != nil {
            return err
        }
        
        // Формируем ответ и отправляем в канал
        response := &model.NotificationResponse{
            Success:      true,
            Message:      utils.T(ctx, "success.notification.new"),
            Notification: notificationEvent,
            // ... дополнительные поля
        }
        
        ch <- response
        return nil
    }
    
    if err := subscriptionService.Subscribe(ctx, channel, eventHandler); err != nil {
        close(ch)
        return nil, err
    }
    
    return ch, nil
}
```

### Пример создания нового резолвера подписки

```go
// Пример для создания подписки на обновления пользователей
func (r *subscriptionResolver) UpdatedUser(ctx context.Context, userID *uuid.UUID) (<-chan *model.UserResponse, error) {
    // 1. Создаем канал для отправки результатов клиенту
    ch := make(chan *model.UserResponse, 1)
    
    // 2. Создаем сервис подписок
    subscriptionService := subscription.New()
    
    // 3. Формируем имя канала на основе типа сущности и ID
    var entityID *string
    if userID != nil {
        idStr := userID.String()
        entityID = &idStr
    }
    
    channel, err := subscriptionService.BuildChannelName(ctx, "user", entityID)
    if err != nil {
        close(ch)
        return nil, err
    }
    
    // 4. Создаем обработчик событий для данного типа подписки
    eventHandler := func(ctx context.Context, payload []byte) error {
        var evt subscription.EntityEvent
        if err := json.Unmarshal(payload, &evt); err != nil {
            return err
        }
        
        // Проверяем, что тип события соответствует пользователю
        if evt.Type != "user" {
            return nil
        }
        
        // Получаем актуальные данные из базы
        qr := &queryResolver{r.Resolver}
        response, err := qr.User(ctx, evt.EntityID)
        if err != nil {
            return err
        }
        
        if response != nil {
            ch <- response
        }
        
        return nil
    }
    
    // 5. Подписываемся на канал с использованием сервиса
    if err := subscriptionService.Subscribe(ctx, channel, eventHandler); err != nil {
        close(ch)
        return nil, err
    }
    
    // 6. Обрабатываем завершение контекста
    go func() {
        <-ctx.Done()
        close(ch)
    }()
    
    return ch, nil
}
```

### Публикация событий

Для публикации событий используйте методы Publisher:

```go
// Пример публикации события об обновлении пользователя
publisher := subscription.NewPublisher()

// Для стандартных событий (создание, обновление, удаление)
if err := publisher.PublishEntityUpdated(ctx, "user", userID); err != nil {
    // Обработка ошибки (логирование ошибки)
}

// Для уведомлений пользователя (специальный метод)
if err := publisher.PublishNotificationEvent(ctx, notificationID, userID, subscription.EntityActionCreated); err != nil {
    // Обработка ошибки (логирование ошибки)
}

// Для событий с дополнительными метаданными
metadata := map[string]any{
    "field": "email",
    "old_value": "old@example.com",
    "new_value": "new@example.com",
}
if err := publisher.PublishEntityEvent(ctx, "user", userID, subscription.EntityActionUpdated, metadata); err != nil {
    // Обработка ошибки (логирование ошибки)
}
```

## GraphQL подписка на уведомления

### Использование на фронтенде

```graphql
# GraphQL подписка на уведомления пользователя
subscription {
  updatedNotification {
    success
    message
    notification {
      id
      title
      content
      isRead
      createTime
      ticket {
        id
        title
        number
      }
    }
    unreadNotificationCount {
      count
      lastUpdated
    }
  }
}
```

### Пример использования в React компоненте

```typescript
import { useSubscription } from '@apollo/client';
import { UPDATED_NOTIFICATION_SUBSCRIPTION } from './graphql/subscriptions';

const NotificationComponent = () => {
  const { data, loading, error } = useSubscription(UPDATED_NOTIFICATION_SUBSCRIPTION);

  useEffect(() => {
    if (data?.updatedNotification) {
      const { notification, unreadNotificationCount } = data.updatedNotification;
      
      // Показать уведомление пользователю
      if (notification && !notification.isRead) {
        showNotification({
          title: notification.title,
          message: notification.content,
          type: 'info'
        });
      }
      
      // Обновить счетчик непрочитанных уведомлений
      updateUnreadCount(unreadNotificationCount.count);
    }
  }, [data]);

  if (loading) return <div>Подключение к уведомлениям...</div>;
  if (error) return <div>Ошибка подключения: {error.message}</div>;

  return (
    <div>
      {/* Ваш компонент уведомлений */}
    </div>
  );
};
```

## Хуки для автоматической публикации

### Хук уведомлений
Реализован хук `NotificationEventCreateHook()` который автоматически публикует события при создании новых уведомлений:

```go
// В схеме NotificationEvent
func (NotificationEvent) Hooks() []ent.Hook {
    return []ent.Hook{
        hooks.NotificationEventCreateHook(),
    }
}
```

Хук автоматически:
1. Отслеживает создание новых записей `NotificationEvent`
2. Получает ID пользователя из связи `user`
3. Публикует событие в канал конкретного пользователя
4. Логирует ошибки без прерывания выполнения основной операции

## Расширение сервиса

При необходимости сервис может быть расширен:

1. Добавлением новых методов для формирования специализированных каналов
2. Поддержкой дополнительных механизмов обработки ошибок
3. Добавлением возможности фильтрации событий
4. Реализацией механизма "heartbeat" для проверки активности подписки

## Периодическая ревалидация WebSocket

При установке WebSocket-соединения токен сессии валидируется. Дополнительно сервер раз в 5 минут выполняет повторную проверку токена:

- Если токен перестал быть валидным (сессия отозвана, истекла, изменился token_version) — соединение закрывается сервером.
- Частота ревалидации настраивается в `server/server.go` (ticker 5m).
- Отпечаток устройства (fingerprint: браузер/ОС по мажорным версиям) также учитывается для пометки сессии `is_high_risk`, но не рвёт соединение автоматически.
