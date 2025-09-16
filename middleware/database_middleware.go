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
	dbClientMutex  sync.RWMutex
)

// InitDatabaseClient initializes the global database client
// Can be called multiple times - will retry if previous attempts failed
func InitDatabaseClient(ctx context.Context) error {
	dbClientMutex.Lock()
	defer dbClientMutex.Unlock()

	// If already initialized successfully, return nil
	if globalDBClient != nil {
		return nil
	}

	config := database.GetConfigFromEnv()
	client, err := database.NewClient(ctx, config)
	if err != nil {
		utils.Logger.Error("Failed to initialize database client",
			zap.Error(err),
		)
		return err
	}

	globalDBClient = client
	utils.Logger.Info("Database client initialized successfully")
	return nil
}

// GetDatabaseClient returns the global database client
func GetDatabaseClient() *database.Client {
	dbClientMutex.RLock()
	defer dbClientMutex.RUnlock()
	return globalDBClient
}

// CloseDatabaseClient closes the global database client
// This should be called during application shutdown
func CloseDatabaseClient() error {
	dbClientMutex.Lock()
	defer dbClientMutex.Unlock()

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
		// Check if client exists (with read lock for performance)
		dbClientMutex.RLock()
		client := globalDBClient
		dbClientMutex.RUnlock()

		// If no client exists, try to initialize it
		if client == nil {
			if err := InitDatabaseClient(r.Context()); err != nil {
				utils.Logger.Error("Database client init failed",
					zap.Error(err),
				)
				http.Error(w, "Database not available", http.StatusServiceUnavailable)
				return
			}

			// Get the client again after successful initialization
			dbClientMutex.RLock()
			client = globalDBClient
			dbClientMutex.RUnlock()
		}

		// Add database client to context using local key
		ctx := context.WithValue(r.Context(), dbContextKey{}, client)

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
