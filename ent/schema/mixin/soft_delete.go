package mixin

import (
	"context"
	"main/ent/intercept"
	"main/utils"

	"entgo.io/contrib/entgql"
	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"
)

// SoftDeleteMixin mixin for soft delete
type SoftDeleteMixin struct {
	mixin.Schema
}

// Fields of the SoftDeleteMixin
func (SoftDeleteMixin) Fields() []ent.Field {
	return []ent.Field{
		field.Time("deleted_at").
			Optional().
			Annotations(
				entgql.Skip(entgql.SkipMutationCreateInput, entgql.SkipMutationUpdateInput),
			),
	}
}

type SoftDeleteKey struct{}

// SkipSoftDelete returns a new context that skips the soft delete interceptor
func SkipSoftDelete(parent context.Context) context.Context {
	return context.WithValue(parent, SoftDeleteKey{}, true)
}

// Interceptors of the SoftDeleteMixin.
func (d SoftDeleteMixin) Interceptors() []ent.Interceptor {
	return []ent.Interceptor{
		intercept.TraverseFunc(func(ctx context.Context, q intercept.Query) error {
			// Skip soft delete, including deleted entities
			if skip, _ := ctx.Value(SoftDeleteKey{}).(bool); skip {
				utils.Logger.Debug("SoftDeleteMixin skipped due to context flag")
				return nil
			}

			// Apply soft delete filter using WhereP
			q.WhereP(sql.FieldIsNull("deleted_at"))

			return nil
		}),
	}
}

// Hooks of the SoftDeleteMixin
func (d SoftDeleteMixin) Hooks() []ent.Hook {
	return []ent.Hook{
		// Note: Ent framework doesn't support changing operation type (Delete -> Update) in hooks.
		// Soft delete is implemented via interceptors for filtering queries.
		// To perform soft delete, use UpdateOneID().SetDeletedAt(time.Now()) instead of DeleteOneID().
	}
}

// P adds a storage-level predicate to queries and mutations
func (d SoftDeleteMixin) P(w interface{ WhereP(...func(*sql.Selector)) }) {
	w.WhereP(
		sql.FieldIsNull("deleted_at"),
	)
}
