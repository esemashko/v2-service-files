package mixin

import (
	"context"
	"main/ent/intercept"

	"entgo.io/ent"
	"entgo.io/ent/schema/mixin"
)

// LimitMixin adds a default limit to queries
type LimitMixin struct {
	mixin.Schema
}

// Interceptors of the LimitMixin.
func (LimitMixin) Interceptors() []ent.Interceptor {
	return []ent.Interceptor{
		intercept.TraverseFunc(func(ctx context.Context, q intercept.Query) error {
			// Limit the number of records returned to 100,
			// in case Limit was not explicitly set.
			if ent.QueryFromContext(ctx).Limit == nil {
				q.Limit(1000)
			}
			return nil
		}),
	}
}
