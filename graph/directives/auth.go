package directives

import (
	"context"
	"main/security"

	"github.com/99designs/gqlgen/graphql"
)

// Auth директива для проверки авторизации по пользователю
func Auth(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
	errMsg := security.ValidateAuthAccess(ctx)
	if errMsg != nil {
		return nil, errMsg
	}

	return next(ctx)
}
