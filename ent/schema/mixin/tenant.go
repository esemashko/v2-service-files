package mixin

import (
	"context"
	"fmt"
	"main/ent/intercept"

	"entgo.io/contrib/entgql"
	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"
	federation "github.com/esemashko/v2-federation"
	"github.com/google/uuid"
)

// TenantMixin добавляет tenant_id и глобальный фильтр по нему
type TenantMixin struct {
	mixin.Schema
}

// Fields of the TenantMixin.
func (TenantMixin) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("tenant_id", uuid.UUID{}).
			Immutable().
			Annotations(
				entgql.Skip(),
			),
	}
}

// TenantFilterKey is used to skip tenant filtering in specific contexts
type TenantFilterKey struct{}

// SkipTenantFilter returns a new context that skips the tenant filter interceptor
func SkipTenantFilter(parent context.Context) context.Context {
	return context.WithValue(parent, TenantFilterKey{}, true)
}

// Interceptors of the TenantMixin for automatic tenant filtering
func (TenantMixin) Interceptors() []ent.Interceptor {
	return []ent.Interceptor{
		// Interceptor for Query operations
		intercept.TraverseFunc(func(ctx context.Context, q intercept.Query) error {
			// Skip tenant filter if explicitly requested
			if skip, _ := ctx.Value(TenantFilterKey{}).(bool); skip {
				return nil
			}

			// Get tenant from context
			tenantID := federation.GetTenantID(ctx)
			if tenantID == nil || *tenantID == uuid.Nil {
				// If no tenant in context, skip filtering (e.g., system operations)
				return nil
			}

			// Apply tenant filter
			q.WhereP(func(s *sql.Selector) {
				s.Where(sql.EQ(s.C("tenant_id"), *tenantID))
			})

			return nil
		}),
	}
}

// Hooks of the TenantMixin
func (TenantMixin) Hooks() []ent.Hook {
	return []ent.Hook{
		func(next ent.Mutator) ent.Mutator {
			return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
				// Skip if explicitly requested
				if skip, _ := ctx.Value(TenantFilterKey{}).(bool); skip {
					return next.Mutate(ctx, m)
				}

				// Get tenant from context
				tenantID := federation.GetTenantID(ctx)

				// Auto-set tenant_id on creation
				if m.Op().Is(ent.OpCreate) {
					if tenantID == nil || *tenantID == uuid.Nil {
						// Tenant is required for creating records (unless explicitly skipped)
						return nil, fmt.Errorf("tenant is required for creating records")
					}
					if err := m.SetField("tenant_id", *tenantID); err != nil {
						// Field might not exist on this entity, ignore error
					}
				}

				// Apply tenant filter for Update and Delete operations
				if m.Op().Is(ent.OpUpdate | ent.OpUpdateOne | ent.OpDelete | ent.OpDeleteOne) {
					if tenantID != nil && *tenantID != uuid.Nil {
						// Add tenant_id filter to the mutation using WhereP if available
						if mutationWithWhere, ok := m.(interface {
							WhereP(...func(*sql.Selector))
						}); ok {
							mutationWithWhere.WhereP(func(s *sql.Selector) {
								s.Where(sql.EQ(s.C("tenant_id"), *tenantID))
							})
						}
					}
				}

				return next.Mutate(ctx, m)
			})
		},
	}
}

// P adds a storage-level predicate to queries and mutations
func (TenantMixin) P(w interface{ WhereP(...func(*sql.Selector)) }) {
	w.WhereP(
		sql.FieldNotNull("tenant_id"),
	)
}
