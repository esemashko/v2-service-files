package websocket

import (
	"context"
	"errors"
	"main/redis"
	"main/utils"

	federation "github.com/esemashko/v2-federation"
	"go.uber.org/zap"
)

// EventHandler определяет тип обработчика событий для универсальной подписки.
// Обработчик принимает контекст и сырой payload (который можно JSON-десериализовать в нужную структуру).
type EventHandler func(ctx context.Context, payload []byte) error

// SubscriptionService инкапсулирует общую бизнес-логику подписок.
type SubscriptionService struct{}

// New создает новый экземпляр сервиса подписок.
func New() *SubscriptionService {
	return &SubscriptionService{}
}

// Subscribe выполняет подписку на указанный channel и вызывает переданный обработчик для каждого сообщения.
// Возвращает канал для отмены подписки (закрытие канала отменяет подписку).
func (s *SubscriptionService) Subscribe(ctx context.Context, channel string, handler EventHandler) error {
	// Проверяем наличие tenant в контексте
	tenantIDPtr := federation.GetTenantID(ctx)
	if tenantIDPtr == nil {
		utils.Logger.Error("Subscription attempt without tenant context")
		return errors.New(utils.T(ctx, "error.unauthorized"))
	}

	tenantID := tenantIDPtr.String()

	// Получаем Redis клиент
	redisService, err := redis.GetTenantCacheService()
	if err != nil || redisService == nil || redisService.GetClient() == nil {
		utils.Logger.Error("Redis unavailable for websocket", zap.Error(err))
		return errors.New(utils.T(ctx, "error.internal.redis_unavailable"))
	}
	redisClient := redisService.GetClient()

	// Подписываемся на канал Redis
	pubsub := redisClient.Subscribe(ctx, channel)
	chEvents := pubsub.Channel()

	// Проверяем, что подписка успешно создана
	if chEvents == nil {
		utils.Logger.Error("Failed to create Redis websocket channel",
			zap.String("tenantID", tenantID),
			zap.String("channel", channel))
		return errors.New(utils.T(ctx, "error.internal.redis_subscription_failed"))
	}

	// Запускаем горутину для обработки сообщений
	go func() {
		var nilMessageCount int // Счетчик последовательных nil сообщений
		defer func() {
			if r := recover(); r != nil {
				utils.Logger.Error("Panic in websocket handler",
					zap.String("tenantID", tenantID),
					zap.String("channel", channel),
					zap.Any("panic", r))
			}
			if err := pubsub.Close(); err != nil {
				utils.Logger.Error("Error closing Redis pubsub",
					zap.String("tenantID", tenantID),
					zap.String("channel", channel),
					zap.Error(err))
			}
			utils.Logger.Info("Subscription ended and cleaned up",
				zap.String("tenantID", tenantID),
				zap.String("channel", channel))
		}()

		for {
			select {
			case <-ctx.Done():
				utils.Logger.Info("Subscription closed (context done)",
					zap.String("tenantID", tenantID),
					zap.String("channel", channel),
					zap.Error(ctx.Err()))
				return
			case msg := <-chEvents:
				// Проверяем, что сообщение не nil (может быть nil при закрытии Redis соединения)
				if msg == nil {
					nilMessageCount++
					utils.Logger.Warn("Received nil message from Redis channel, connection may be closed",
						zap.String("tenantID", tenantID),
						zap.String("channel", channel),
						zap.Int("consecutive_nil_count", nilMessageCount))

					// Если получили несколько nil сообщений подряд, считаем канал закрытым
					if nilMessageCount >= 3 {
						utils.Logger.Info("Redis channel closed after multiple nil messages, ending websocket",
							zap.String("tenantID", tenantID),
							zap.String("channel", channel),
							zap.Int("nil_count", nilMessageCount))
						return
					}
					continue
				}

				// Сбрасываем счетчик nil сообщений при получении валидного сообщения
				nilMessageCount = 0

				// Вызываем обработчик для обработки события
				if err := handler(ctx, []byte(msg.Payload)); err != nil {
					utils.Logger.Error("Error handling websocket event",
						zap.String("tenantID", tenantID),
						zap.String("channel", channel),
						zap.Error(err))
				}
			}
		}
	}()

	return nil
}

// BuildChannelName формирует имя канала на основе tenantID, типа сущности и идентификатора
func (s *SubscriptionService) BuildChannelName(ctx context.Context, entityType string, entityID *string) (string, error) {
	tenantIDPtr := federation.GetTenantID(ctx)
	if tenantIDPtr == nil {
		return "", errors.New(utils.T(ctx, "error.unauthorized"))
	}

	tenantID := tenantIDPtr.String()

	if entityID != nil {
		return tenantID + ":" + entityType + "_" + *entityID, nil
	}

	return tenantID + ":" + entityType + ":updates", nil
}
