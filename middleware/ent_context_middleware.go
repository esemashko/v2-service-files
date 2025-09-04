package middleware

import (
	"context"
	"main/ent"

	"github.com/99designs/gqlgen/graphql"
)

// EntContextMiddleware добавляет Ent клиент в контекст для всех GraphQL операций
func EntContextMiddleware(client *ent.Client) graphql.OperationMiddleware {
	return func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
		// Добавляем клиент в контекст
		ctxWithClient := ent.NewContext(ctx, client)
		return next(ctxWithClient)
	}
}
