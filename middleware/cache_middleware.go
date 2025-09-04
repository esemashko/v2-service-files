package middleware

import (
	"context"
	"main/database"
	"main/utils"

	"github.com/99designs/gqlgen/graphql"
	federation "github.com/esemashko/v2-federation"
	"github.com/vektah/gqlparser/v2/ast"
	"go.uber.org/zap"
)

// GraphQLCacheMiddleware для автоматического управления кешем GraphQL операций
func GraphQLCacheMiddleware() graphql.OperationMiddleware {
	return func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
		opCtx := graphql.GetOperationContext(ctx)
		if opCtx != nil && opCtx.Operation != nil {
			op := opCtx.Operation.Operation

			// Устанавливаем режим кеширования в зависимости от типа операции
			switch op {
			case ast.Query:
				// Кеш разрешен только при наличии тенанта, иначе полностью отключаем
				if federation.GetTenantID(ctx) != nil {
					ctx = database.EnableContextCache(ctx)
				} else {
					ctx = database.SkipCache(ctx)
				}
			case ast.Mutation:
				// Пропускаем кэширование для мутаций
				ctx = database.SkipCache(ctx)
			case ast.Subscription:
				// Не включаем контекстный кэш для подписок, чтобы избежать stale данных в долгоживущих сессиях
			}

			if database.IsDebugDB() {
				utils.Logger.Debug("GraphQL operation cache control applied",
					zap.String("operation_name", opCtx.OperationName),
					zap.String("operation_type", string(op)),
					zap.Bool("tenant_present", federation.GetTenantID(ctx) != nil),
					zap.Bool("cache_enabled", op == ast.Query && federation.GetTenantID(ctx) != nil),
				)
			}
		}

		return next(ctx)
	}
}
