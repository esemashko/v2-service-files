package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"main/redis"
	"main/utils"
	"time"

	federation "github.com/esemashko/v2-federation"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Publisher предоставляет методы для публикации событий в Redis Pub/Sub
type Publisher struct {
	subscriptionService *SubscriptionService
}

// NewPublisher создает новый экземпляр публикатора событий
func NewPublisher() *Publisher {
	return &Publisher{
		subscriptionService: New(),
	}
}

// PublishEntityUpdated публикует событие обновления сущности
func (p *Publisher) PublishEntityUpdated(ctx context.Context, entityType string, entityID uuid.UUID) error {
	tenantIDPtr := federation.GetTenantID(ctx)
	if tenantIDPtr == nil {
		return errors.New(utils.T(ctx, "error.unauthorized"))
	}

	// Формируем каналы публикации
	channels := make([]string, 0, 2)

	// Для ticket и ticket_comment публикуем также в глобальный канал списка
	if entityType == "ticket" || entityType == "ticket_comment" {
		ch, err := p.subscriptionService.BuildChannelName(ctx, entityType, nil)
		if err != nil {
			return err
		}
		channels = append(channels, ch)
	}

	// Для всех типов, кроме ticket_comment, публикуем в канал конкретной сущности
	if entityType != "ticket_comment" {
		idStr := entityID.String()
		ch, err := p.subscriptionService.BuildChannelName(ctx, entityType, &idStr)
		if err != nil {
			return err
		}
		channels = append(channels, ch)
	}

	// Создаем событие
	event := EntityEvent{
		Action:   EntityActionUpdated,
		EntityID: entityID,
		Type:     entityType,
	}

	for _, ch := range channels {
		if err := p.publishEvent(ctx, ch, event); err != nil {
			return err
		}
	}
	return nil
}

// PublishEntityDeleted публикует событие удаления сущности
func (p *Publisher) PublishEntityDeleted(ctx context.Context, entityType string, entityID uuid.UUID) error {
	tenantIDPtr := federation.GetTenantID(ctx)
	if tenantIDPtr == nil {
		return errors.New(utils.T(ctx, "error.unauthorized"))
	}

	channels := make([]string, 0, 2)

	if entityType == "ticket" || entityType == "ticket_comment" {
		ch, err := p.subscriptionService.BuildChannelName(ctx, entityType, nil)
		if err != nil {
			return err
		}
		channels = append(channels, ch)
	}

	if entityType != "ticket_comment" {
		idStr := entityID.String()
		ch, err := p.subscriptionService.BuildChannelName(ctx, entityType, &idStr)
		if err != nil {
			return err
		}
		channels = append(channels, ch)
	}

	event := EntityEvent{
		Action:   EntityActionDeleted,
		EntityID: entityID,
		Type:     entityType,
	}

	for _, ch := range channels {
		if err := p.publishEvent(ctx, ch, event); err != nil {
			return err
		}
	}
	return nil
}

// PublishEntityCreated публикует событие создания сущности
func (p *Publisher) PublishEntityCreated(ctx context.Context, entityType string, entityID uuid.UUID) error {
	tenantIDPtr := federation.GetTenantID(ctx)
	if tenantIDPtr == nil {
		return errors.New(utils.T(ctx, "error.unauthorized"))
	}

	// Формируем канал для глобальных обновлений по типу сущности
	channel, err := p.subscriptionService.BuildChannelName(ctx, entityType, nil)
	if err != nil {
		return err
	}

	// Создаем событие
	event := EntityEvent{
		Action:   EntityActionCreated,
		EntityID: entityID,
		Type:     entityType,
	}

	return p.publishEvent(ctx, channel, event)
}

// PublishEntityEvent публикует произвольное событие с дополнительными метаданными
func (p *Publisher) PublishEntityEvent(ctx context.Context, entityType string, entityID uuid.UUID, action EntityAction, metadata map[string]any) error {
	tenantIDPtr := federation.GetTenantID(ctx)
	if tenantIDPtr == nil {
		return errors.New(utils.T(ctx, "error.unauthorized"))
	}

	// Формируем имя канала для подписки
	idStr := entityID.String()
	channel, err := p.subscriptionService.BuildChannelName(ctx, entityType, &idStr)
	if err != nil {
		return err
	}

	// Создаем событие
	event := EntityEvent{
		Action:   action,
		EntityID: entityID,
		Type:     entityType,
		Metadata: metadata,
	}

	return p.publishEvent(ctx, channel, event)
}

// PublishMessageEvent публикует событие сообщения в глобальный канал
func (p *Publisher) PublishMessageEvent(ctx context.Context, messageID uuid.UUID, action EntityAction) error {
	tenantIDPtr := federation.GetTenantID(ctx)
	if tenantIDPtr == nil {
		return errors.New(utils.T(ctx, "error.unauthorized"))
	}

	// Для сообщений всегда используем глобальный канал
	channel, err := p.subscriptionService.BuildChannelName(ctx, "message", nil)
	if err != nil {
		return err
	}

	// Создаем событие
	event := EntityEvent{
		Action:   action,
		EntityID: messageID,
		Type:     "message",
	}

	utils.Logger.Debug("Publishing message event",
		zap.String("channel", channel),
		zap.String("action", string(action)),
		zap.String("messageID", messageID.String()))

	return p.publishEvent(ctx, channel, event)
}

// PublishMessageEventToChat публикует событие сообщения для конкретного чата
func (p *Publisher) PublishMessageEventToChat(ctx context.Context, messageID uuid.UUID, chatID uuid.UUID, action EntityAction) error {
	tenantIDPtr := federation.GetTenantID(ctx)
	if tenantIDPtr == nil {
		return errors.New(utils.T(ctx, "error.unauthorized"))
	}

	// Формируем канал для сообщений конкретного чата
	chatIDStr := chatID.String()
	channel, err := p.subscriptionService.BuildChannelName(ctx, "message_chat", &chatIDStr)
	if err != nil {
		return err
	}

	event := EntityEvent{
		Action:   action,
		EntityID: messageID,
		Type:     "message",
		Metadata: map[string]any{
			"chat_id": chatID.String(),
		},
	}

	utils.Logger.Debug("Publishing message event to chat",
		zap.String("channel", channel),
		zap.String("action", string(action)),
		zap.String("messageID", messageID.String()),
		zap.String("chatID", chatID.String()))

	return p.publishEvent(ctx, channel, event)
}

// PublishOnlineStatusEvent публикует событие изменения онлайн статуса пользователя
func (p *Publisher) PublishOnlineStatusEvent(ctx context.Context, userID uuid.UUID, isOnline bool) error {
	tenantIDPtr := federation.GetTenantID(ctx)
	if tenantIDPtr == nil {
		return errors.New(utils.T(ctx, "error.unauthorized"))
	}

	// Формируем канал для онлайн статуса
	channel, err := p.subscriptionService.BuildChannelName(ctx, "online_status", nil)
	if err != nil {
		return err
	}

	// Создаем событие
	// Всегда используем Updated, так как это изменение статуса (online/offline)
	event := EntityEvent{
		Action:   EntityActionUpdated,
		EntityID: userID,
		Type:     "user_online_status",
		Metadata: map[string]any{
			"is_online": isOnline,
			"timestamp": time.Now(),
		},
	}

	utils.Logger.Debug("Publishing online status event",
		zap.String("channel", channel),
		zap.String("userID", userID.String()),
		zap.Bool("isOnline", isOnline))

	return p.publishEvent(ctx, channel, event)
}

// PublishNotificationEvent публикует событие уведомления для конкретного пользователя
func (p *Publisher) PublishNotificationEvent(ctx context.Context, notificationID uuid.UUID, userID uuid.UUID, action EntityAction) error {
	tenantIDPtr := federation.GetTenantID(ctx)
	if tenantIDPtr == nil {
		return errors.New(utils.T(ctx, "error.unauthorized"))
	}

	// Формируем канал для уведомлений конкретного пользователя
	userIDStr := userID.String()
	channel, err := p.subscriptionService.BuildChannelName(ctx, "notification_user", &userIDStr)
	if err != nil {
		return err
	}

	// Создаем событие
	event := EntityEvent{
		Action:   action,
		EntityID: notificationID,
		Type:     "notification",
		Metadata: map[string]any{
			"user_id": userID,
		},
	}

	utils.Logger.Info("Publishing notification event",
		zap.String("channel", channel),
		zap.String("notification_id", notificationID.String()),
		zap.String("user_id", userID.String()),
		zap.String("action", string(action)))

	return p.publishEvent(ctx, channel, event)
}

// PublishTicketWorkTimeEvent публикует событие изменения учета времени для тикета
func (p *Publisher) PublishTicketWorkTimeEvent(ctx context.Context, ticketID uuid.UUID, workTimeID uuid.UUID, action EntityAction) error {
	tenantIDPtr := federation.GetTenantID(ctx)
	if tenantIDPtr == nil {
		return errors.New(utils.T(ctx, "error.unauthorized"))
	}

	// Формируем канал для событий учета времени конкретного тикета
	ticketIDStr := ticketID.String()
	channel, err := p.subscriptionService.BuildChannelName(ctx, "ticket_work_time", &ticketIDStr)
	if err != nil {
		return err
	}

	// Создаем событие
	event := EntityEvent{
		Action:   action,
		EntityID: ticketID,
		Type:     "ticket_work_time",
		Metadata: map[string]any{
			"work_time_id": workTimeID.String(),
			"ticket_id":    ticketID.String(),
		},
	}

	utils.Logger.Debug("Publishing ticket work time event",
		zap.String("channel", channel),
		zap.String("ticket_id", ticketID.String()),
		zap.String("work_time_id", workTimeID.String()),
		zap.String("action", string(action)))

	return p.publishEvent(ctx, channel, event)
}

// publishEvent приватный метод для публикации события в Redis
func (p *Publisher) publishEvent(ctx context.Context, channel string, event interface{}) error {
	// Получаем Redis клиент
	redisService, err := redis.GetTenantCacheService()
	if err != nil || redisService == nil || redisService.GetClient() == nil {
		utils.Logger.Error("Redis unavailable for event publishing", zap.Error(err))
		return errors.New(utils.T(ctx, "error.internal.redis_unavailable"))
	}
	redisClient := redisService.GetClient()

	// Сериализуем событие
	eventJSON, err := json.Marshal(event)
	if err != nil {
		utils.Logger.Error("Failed to marshal event", zap.Error(err), zap.Any("event", event))
		return err
	}

	// Публикуем событие
	if err := redisClient.Publish(ctx, channel, eventJSON).Err(); err != nil {
		utils.Logger.Error("Failed to publish event",
			zap.Error(err),
			zap.String("channel", channel),
			zap.Any("event", event))
		return err
	}

	utils.Logger.Debug("Successfully published event to Redis",
		zap.String("channel", channel),
		zap.String("eventJSON", string(eventJSON)))

	return nil
}
