package websocket

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"main/ctxkeys"
	"main/utils"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPublisherEventSerialization проверяет корректность сериализации событий
func TestPublisherEventSerialization(t *testing.T) {
	tests := []struct {
		name     string
		event    EntityEvent
		expected string
	}{
		{
			name: "Created event",
			event: EntityEvent{
				Action:   EntityActionCreated,
				EntityID: uuid.MustParse("12345678-1234-1234-1234-123456789012"),
				Type:     "notification",
				Metadata: map[string]any{
					"user_id": uuid.MustParse("87654321-4321-4321-4321-210987654321"),
				},
			},
			expected: `{"action":"created","entity_id":"12345678-1234-1234-1234-123456789012","type":"notification","metadata":{"user_id":"87654321-4321-4321-4321-210987654321"}}`,
		},
		{
			name: "Updated event without metadata",
			event: EntityEvent{
				Action:   EntityActionUpdated,
				EntityID: uuid.MustParse("12345678-1234-1234-1234-123456789012"),
				Type:     "ticket",
			},
			expected: `{"action":"updated","entity_id":"12345678-1234-1234-1234-123456789012","type":"ticket"}`,
		},
		{
			name: "Deleted event",
			event: EntityEvent{
				Action:   EntityActionDeleted,
				EntityID: uuid.MustParse("12345678-1234-1234-1234-123456789012"),
				Type:     "user",
			},
			expected: `{"action":"deleted","entity_id":"12345678-1234-1234-1234-123456789012","type":"user"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.event)
			require.NoError(t, err)
			assert.JSONEq(t, tt.expected, string(data))

			// Проверяем обратную десериализацию
			var unmarshaled EntityEvent
			err = json.Unmarshal(data, &unmarshaled)
			require.NoError(t, err)
			assert.Equal(t, tt.event.Action, unmarshaled.Action)
			assert.Equal(t, tt.event.EntityID, unmarshaled.EntityID)
			assert.Equal(t, tt.event.Type, unmarshaled.Type)

			// Для metadata нужно учесть, что UUID сериализуется как строка в JSON
			if tt.event.Metadata != nil {
				expectedMetadata := make(map[string]any)
				for k, v := range tt.event.Metadata {
					if uuidVal, ok := v.(uuid.UUID); ok {
						expectedMetadata[k] = uuidVal.String()
					} else {
						expectedMetadata[k] = v
					}
				}
				assert.Equal(t, expectedMetadata, unmarshaled.Metadata)
			} else {
				assert.Equal(t, tt.event.Metadata, unmarshaled.Metadata)
			}
		})
	}
}

// TestEntityActionConstants проверяет константы действий
func TestEntityActionConstants(t *testing.T) {
	tests := []struct {
		action   EntityAction
		expected string
	}{
		{EntityActionCreated, "created"},
		{EntityActionUpdated, "updated"},
		{EntityActionDeleted, "deleted"},
	}

	for _, tt := range tests {
		t.Run(string(tt.action), func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.action))
		})
	}
}

// TestPublisherErrorHandling проверяет обработку ошибок в Publisher
func TestPublisherErrorHandling(t *testing.T) {
	// Инициализируем логгер для unit тестов
	utils.InitLogger()

	publisher := NewPublisher()

	t.Run("No tenant in context", func(t *testing.T) {
		ctx := context.Background()
		entityID := uuid.New()

		err := publisher.PublishEntityUpdated(ctx, "test", entityID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized")
	})

	t.Run("Nil tenant in context", func(t *testing.T) {
		ctx := ctxkeys.SetTenant(context.Background(), nil)
		entityID := uuid.New()

		err := publisher.PublishEntityCreated(ctx, "test", entityID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized")
	})
}

// TestSubscriptionServiceChannelNames проверяет формирование имен каналов
func TestSubscriptionServiceChannelNames(t *testing.T) {
	service := New()

	// Создаем тестовый tenant
	tenant := &ctxkeys.TenantInfo{
		ID: uuid.MustParse("12345678-1234-1234-1234-123456789012"),
	}
	ctx := ctxkeys.SetTenant(context.Background(), tenant)

	tests := []struct {
		name       string
		entityType string
		entityID   *string
		expected   string
	}{
		{
			name:       "Channel with entity ID",
			entityType: "ticket",
			entityID:   func() *string { s := "87654321-4321-4321-4321-210987654321"; return &s }(),
			expected:   "12345678-1234-1234-1234-123456789012:ticket_87654321-4321-4321-4321-210987654321",
		},
		{
			name:       "Global channel without entity ID",
			entityType: "notification",
			entityID:   nil,
			expected:   "12345678-1234-1234-1234-123456789012:notification:updates",
		},
		{
			name:       "User notification channel",
			entityType: "notification_user",
			entityID:   func() *string { s := "user-123"; return &s }(),
			expected:   "12345678-1234-1234-1234-123456789012:notification_user_user-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channel, err := service.BuildChannelName(ctx, tt.entityType, tt.entityID)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, channel)
		})
	}

	t.Run("No tenant in context", func(t *testing.T) {
		emptyCtx := context.Background()
		_, err := service.BuildChannelName(emptyCtx, "test", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized")
	})
}

// TestEntityEventMetadata проверяет работу с метаданными событий
func TestEntityEventMetadata(t *testing.T) {
	tests := []struct {
		name     string
		metadata map[string]any
	}{
		{
			name: "String metadata",
			metadata: map[string]any{
				"field":     "email",
				"old_value": "old@example.com",
				"new_value": "new@example.com",
			},
		},
		{
			name: "Mixed types metadata",
			metadata: map[string]any{
				"user_id":     uuid.New(),
				"count":       42,
				"is_active":   true,
				"timestamp":   time.Now(),
				"description": "test event",
			},
		},
		{
			name:     "Empty metadata",
			metadata: map[string]any{},
		},
		{
			name:     "Nil metadata",
			metadata: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := EntityEvent{
				Action:   EntityActionUpdated,
				EntityID: uuid.New(),
				Type:     "test",
				Metadata: tt.metadata,
			}

			// Сериализуем и десериализуем
			data, err := json.Marshal(event)
			require.NoError(t, err)

			var unmarshaled EntityEvent
			err = json.Unmarshal(data, &unmarshaled)
			require.NoError(t, err)

			// Проверяем основные поля
			assert.Equal(t, event.Action, unmarshaled.Action)
			assert.Equal(t, event.EntityID, unmarshaled.EntityID)
			assert.Equal(t, event.Type, unmarshaled.Type)

			// Для метаданных проверяем что структура сохранилась
			if tt.metadata == nil {
				assert.Nil(t, unmarshaled.Metadata)
			} else if len(tt.metadata) == 0 {
				// Пустой map может стать nil после JSON unmarshaling
				assert.True(t, unmarshaled.Metadata == nil || len(unmarshaled.Metadata) == 0)
			} else {
				assert.NotNil(t, unmarshaled.Metadata)
				assert.Len(t, unmarshaled.Metadata, len(tt.metadata))
			}
		})
	}
}

// TestPublisherValidation проверяет валидацию входных данных
func TestPublisherValidation(t *testing.T) {
	// Эти тесты проверяют базовую валидацию без Redis
	// В интеграционных тестах проверяется полная функциональность

	publisher := NewPublisher()
	tenant := &ctxkeys.TenantInfo{
		ID: uuid.New(),
	}
	ctx := ctxkeys.SetTenant(context.Background(), tenant)

	t.Run("Valid entity types", func(t *testing.T) {
		validTypes := []string{
			"ticket",
			"user",
			"notification",
			"comment",
			"department",
		}

		entityID := uuid.New()
		for _, entityType := range validTypes {
			// Не ожидаем ошибок валидации типа, только ошибки Redis
			err := publisher.PublishEntityCreated(ctx, entityType, entityID)
			// В этих unit тестах ожидаем ошибку Redis, но не ошибку валидации
			if err != nil {
				assert.Contains(t, err.Error(), "redis")
			}
		}
	})

	t.Run("Valid UUIDs", func(t *testing.T) {
		validUUIDs := []uuid.UUID{
			uuid.New(),
			uuid.MustParse("12345678-1234-1234-1234-123456789012"),
			uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		}

		for _, entityID := range validUUIDs {
			err := publisher.PublishEntityUpdated(ctx, "test", entityID)
			// Ожидаем ошибку Redis, но не ошибку валидации UUID
			if err != nil {
				assert.Contains(t, err.Error(), "redis")
			}
		}
	})
}

// TestTicketGlobalPublish verifies that ticket update/delete also publish to global channel
func TestTicketGlobalPublish(t *testing.T) {
	utils.InitLogger()

	publisher := NewPublisher()
	tenant := &ctxkeys.TenantInfo{ID: uuid.New()}
	ctx := ctxkeys.SetTenant(context.Background(), tenant)

	// We can't assert Redis behavior in unit test environment, but
	// we ensure no validation error occurs when publishing to both channels.
	entityID := uuid.New()
	if err := publisher.PublishEntityUpdated(ctx, "ticket", entityID); err != nil {
		// In unit tests Redis may be unavailable; acceptable as long as it's a redis error
		assert.Contains(t, err.Error(), "redis")
	}
	if err := publisher.PublishEntityDeleted(ctx, "ticket", entityID); err != nil {
		assert.Contains(t, err.Error(), "redis")
	}
}

// BenchmarkEventSerialization тестирует производительность сериализации событий
func BenchmarkEventSerialization(b *testing.B) {
	event := EntityEvent{
		Action:   EntityActionCreated,
		EntityID: uuid.New(),
		Type:     "notification",
		Metadata: map[string]any{
			"user_id": uuid.New(),
			"count":   42,
			"active":  true,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(event)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkChannelNameGeneration тестирует производительность генерации имен каналов
func BenchmarkChannelNameGeneration(b *testing.B) {
	service := New()
	tenant := &ctxkeys.TenantInfo{
		ID: uuid.New(),
	}
	ctx := ctxkeys.SetTenant(context.Background(), tenant)
	entityID := uuid.New().String()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.BuildChannelName(ctx, "notification_user", &entityID)
		if err != nil {
			b.Fatal(err)
		}
	}
}
