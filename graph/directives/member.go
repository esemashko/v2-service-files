package directives

import (
	"context"
	"main/security"

	"github.com/99designs/gqlgen/graphql"
)

// Member директива для проверки на роль сотрудника и выше
func Member(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
	errMsg := security.ValidateAdminAccess(ctx)
	if errMsg != nil {
		return nil, errMsg
	}

	return next(ctx)
}
