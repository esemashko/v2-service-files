package middleware

import (
	"context"
	"main/database"
	"main/utils"
	"net/http"
	"sync"

	"go.uber.org/zap"
)

type dbContextKey struct{}

var (
	// Global database client instance
	globalDBClient *database.Client
	dbClientOnce   sync.Once
	dbClientErr    error
)

// InitDatabaseClient initializes the global database client
// This should be called once; can be invoked from middleware lazily
func InitDatabaseClient(ctx context.Context) error {
	dbClientOnce.Do(func() {
		config := database.GetConfigFromEnv()
		globalDBClient, dbClientErr = database.NewClient(ctx, config)
		if dbClientErr != nil {
			utils.Logger.Error("Failed to initialize database client",
				zap.Error(dbClientErr),
			)
		} else {
			utils.Logger.Info("Database client initialized successfully")
		}
	})
	return dbClientErr
}

// GetDatabaseClient returns the global database client
func GetDatabaseClient() *database.Client {
	return globalDBClient
}

// CloseDatabaseClient closes the global database client
// This should be called during application shutdown
func CloseDatabaseClient() error {
	if globalDBClient != nil {
		err := globalDBClient.Close()
		if err != nil {
			utils.Logger.Error("Failed to close database client",
				zap.Error(err),
			)
			return err
		}
		globalDBClient = nil
		utils.Logger.Info("Database client closed successfully")
	}
	return nil
}

// DatabaseMiddleware provides database client in request context
// It should be placed after HeadersMiddleware in the middleware chain
func DatabaseMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Lazily initialize the DB client on first request
		if globalDBClient == nil {
			if err := InitDatabaseClient(r.Context()); err != nil {
				utils.Logger.Error("Database client init failed",
					zap.Error(err),
				)
				http.Error(w, "Database not available", http.StatusServiceUnavailable)
				return
			}
		}

		// Add database client to context using local key
		ctx := context.WithValue(r.Context(), dbContextKey{}, globalDBClient)

		// Call next handler with updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetDBFromContext retrieves the database client from context
func GetDBFromContext(ctx context.Context) *database.Client {
	if client, ok := ctx.Value(dbContextKey{}).(*database.Client); ok {
		return client
	}
	return nil
}
