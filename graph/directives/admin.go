package directives

import (
	"context"
	"main/security"

	"github.com/99designs/gqlgen/graphql"
)

// Admin директива для проверки на администратора и выше
func Admin(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
	errMsg := security.ValidateAdminAccess(ctx)
	if errMsg != nil {
		return nil, errMsg
	}

	return next(ctx)
}
