package websocket

import (
	"github.com/google/uuid"
)

// EntityAction представляет тип действия с сущностью
type EntityAction string

// Константы для действий с сущностями
const (
	EntityActionCreated EntityAction = "created"
	EntityActionUpdated EntityAction = "updated"
	EntityActionDeleted EntityAction = "deleted"
)

// EntityEvent представляет универсальное событие для любой сущности в системе
type EntityEvent struct {
	// Action определяет тип события: created, updated, deleted, etc.
	Action EntityAction `json:"action"`

	// ID сущности, к которой относится событие
	EntityID uuid.UUID `json:"entity_id"`

	// Type определяет тип сущности: ticket, user, etc.
	Type string `json:"type"`

	// Metadata содержит дополнительные данные о событии
	Metadata map[string]any `json:"metadata,omitempty"`
}
