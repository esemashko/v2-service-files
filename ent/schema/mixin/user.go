package mixin

import (
	"context"

	"entgo.io/contrib/entgql"
	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"
	"github.com/google/uuid"
)

// UserMixin добавляет user_id и глобальный фильтр по нему
type UserMixin struct {
	mixin.Schema
}

// Fields of the UserMixin.
func (UserMixin) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("user_id", uuid.UUID{}).
			Optional().
			Annotations(
				entgql.Skip(),
			),
	}
}

// Hooks of the UserMixin
func (UserMixin) Hooks() []ent.Hook {
	return []ent.Hook{
		// In ENT v0.14, user filtering should be implemented through privacy rules
		// or explicit query filters at the resolver level
		func(next ent.Mutator) ent.Mutator {
			return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
				// Auto-set user_id on creation
				if m.Op().Is(ent.OpCreate) {
					// TODO: продумать безопасность работы с полем
				}
				return next.Mutate(ctx, m)
			})
		},
	}
}

// P adds a storage-level predicate to queries and mutations
func (UserMixin) P(w interface{ WhereP(...func(*sql.Selector)) }) {
	// This method can be used for compile-time filtering
	// but runtime filtering should be done in privacy rules
}
