package server

import (
	"context"
	"main/database"
	"main/ent"
	"main/graph/dataloader"
	"main/graph/resolvers"
	"main/middleware"
	"main/utils"
	"net/http"
	"os"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	federation "github.com/esemashko/v2-federation"
	"github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"go.uber.org/zap"
)

// LoggingMiddleware логирует операции GraphQL
func LoggingMiddleware() graphql.OperationMiddleware {
	return func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
		opCtx := graphql.GetOperationContext(ctx)
		utils.Logger.Info("GraphQL operation",
			zap.String("operation_name", opCtx.OperationName),
			zap.String("operation_type", string(opCtx.Operation.Operation)),
		)
		return next(ctx)
	}
}

// NewGraphQLServer creates a new GraphQL server (per request) and selects ent client by operation type
func NewGraphQLServer(db *database.Client) *handler.Server {
	// Базовый клиент для схемы — Query
	srv := handler.New(resolvers.NewSchema(db.Query()))
	if os.Getenv("ENV") != "production" {
		srv.Use(extension.Introspection{})
	}

	// Добавляем HTTP транспорты
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.MultipartForm{
		MaxMemory:     32 << 20,  // 32MB
		MaxUploadSize: 100 << 20, // 100MB
	})

	// Выбор клиента по типу операции и инъекция в контекст
	srv.AroundOperations(func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
		opCtx := graphql.GetOperationContext(ctx)
		if opCtx != nil && opCtx.Operation != nil {
			var entClient *ent.Client
			switch opCtx.Operation.Operation {
			case ast.Query:
				entClient = db.Query()
			case ast.Mutation, ast.Subscription:
				entClient = db.Mutation()
			default:
				entClient = db.Query()
			}

			ctx = ent.NewContext(ctx, entClient)

			// Инициализируем DataLoader и PreloadCache для Query/Mutation (подписки без PreloadCache)
			switch opCtx.Operation.Operation {
			case ast.Query, ast.Mutation:
				loaders := dataloader.NewLoaders(entClient)
				ctx = dataloader.WithLoaders(ctx, loaders)
				cache := dataloader.GetPreloadCache(ctx)
				ctx = dataloader.WithPreloadCache(ctx, cache)
			case ast.Subscription:
				// Для подписок не добавляем PreloadCache (долгоживущие контексты)
				loaders := dataloader.NewLoaders(entClient)
				ctx = dataloader.WithLoaders(ctx, loaders)
			}
		}
		return next(ctx)
	})

	// Cache control per operation type (query vs mutation)
	srv.AroundOperations(middleware.GraphQLCacheMiddleware())

	// Logging
	srv.AroundOperations(LoggingMiddleware())

	return srv
}

func SetupRouter() (*chi.Mux, error) {
	r := chi.NewRouter()

	// i18n initialization
	bundle, err := InitI18n()
	if err != nil {
		return nil, err
	}
	// Устанавливаем глобальный bundle для локализации
	utils.SetI18nBundle(bundle)

	// Global CORS middleware
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   federation.CORSAllowedHeaders,
		ExposedHeaders:   []string{"Link", "X-Request-Id"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Group(func(r chi.Router) {
		r.Use(middleware.DatabaseMiddleware)
		// r.Use(HTTPHeadersLoggingMiddleware)
		r.Use(middleware.FederationMiddleware)

		// Playground только для не-продакшн окружения
		if os.Getenv("ENV") != "production" {
			r.Handle("/", playground.Handler("GraphQL playground", "/query"))
		}

		// Обработчик GraphQL запросов (динамически создаем сервер на каждый запрос)
		r.HandleFunc("/query", func(w http.ResponseWriter, r *http.Request) {
			// Получаем client БД из контекста запроса
			db := middleware.GetDBFromContext(r.Context())
			if db == nil {
				utils.Logger.Error("Database client not found in context")
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			graphqlServer := NewGraphQLServer(db)
			graphqlServer.ServeHTTP(w, r)
		})
	})

	return r, nil
}
