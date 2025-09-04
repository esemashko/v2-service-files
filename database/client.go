package database

import (
	"context"
	"fmt"
	"main/ent"
	"main/redis"
	"main/utils"
	"os"
	"strconv"
	"time"

	"ariga.io/entcache"
	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
)

// Config holds database configuration
type Config struct {
	// Connection endpoints
	QueryDSN    string // Read-only endpoint for queries
	MutationDSN string // Write endpoint for mutations

	// Debug mode
	Debug bool

	// Cache settings (context-level cache for queries)
	EnableCache bool          // Enable context-level caching for query client
	CacheTTL    time.Duration // Cache TTL
}

// Client manages database connections
type Client struct {
	queryClient    *ent.Client
	mutationClient *ent.Client
	config         *Config
}

// GetConfigFromEnv creates config from environment variables
func GetConfigFromEnv() *Config {
	// Base connection parameters
	user := os.Getenv("DB_USER")
	if user == "" {
		user = "postgres"
	}

	password := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "postgres"
	}

	sslMode := os.Getenv("DB_SSLMODE")
	if sslMode == "" {
		sslMode = "disable"
	}

	// Query endpoint (for read operations)
	queryHost := os.Getenv("DB_QUERY_HOST")
	if queryHost == "" {
		queryHost = "localhost"
	}

	queryPort := os.Getenv("DB_QUERY_PORT")
	if queryPort == "" {
		queryPort = "5432"
	}

	// Mutation endpoint (for write operations)
	mutationHost := os.Getenv("DB_MUTATION_HOST")
	if mutationHost == "" {
		mutationHost = "localhost"
	}

	mutationPort := os.Getenv("DB_MUTATION_PORT")
	if mutationPort == "" {
		mutationPort = "5432"
	}

	// Debug mode
	debug, _ := strconv.ParseBool(os.Getenv("DEBUG_DB"))

	// Cache settings
	enableCache, _ := strconv.ParseBool(os.Getenv("ENABLE_DB_CACHE"))
	if enableCache == false {
		enableCache = true // По умолчанию включаем кэш
	}

	cacheTTL := 5 * time.Minute // Значение по умолчанию
	if ttlStr := os.Getenv("DB_CACHE_TTL"); ttlStr != "" {
		if ttlSec, err := strconv.Atoi(ttlStr); err == nil {
			cacheTTL = time.Duration(ttlSec) * time.Second
		}
	}

	// Build DSNs using pgx format
	// Default schema (search_path)
	schema := os.Getenv("DB_SCHEMA")
	if schema == "" {
		schema = "app"
	}

	queryDSN := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s&search_path=%s",
		user, password, queryHost, queryPort, dbName, sslMode, schema,
	)

	mutationDSN := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s&search_path=%s",
		user, password, mutationHost, mutationPort, dbName, sslMode, schema,
	)

	return &Config{
		QueryDSN:    queryDSN,
		MutationDSN: mutationDSN,
		Debug:       debug,
		EnableCache: enableCache,
		CacheTTL:    cacheTTL,
	}
}

// NewClient creates a new database client with separate query and mutation connections
func NewClient(ctx context.Context, config *Config) (*Client, error) {
	if config == nil {
		config = GetConfigFromEnv()
	}

	client := &Client{config: config}

	// Create query client (read-only) with caching
	queryClient, err := createEntClient(ctx, config.QueryDSN, config.Debug, "query", config.EnableCache, config.CacheTTL)
	if err != nil {
		return nil, fmt.Errorf("failed to create query client: %w", err)
	}
	client.queryClient = queryClient

	// Create mutation client (write) without caching but with cache invalidation hook
	mutationClient, err := createEntClient(ctx, config.MutationDSN, config.Debug, "mutation", false, 0)
	if err != nil {
		// Close query client if mutation client fails
		_ = queryClient.Close()
		return nil, fmt.Errorf("failed to create mutation client: %w", err)
	}

	client.mutationClient = mutationClient

	utils.Logger.Info("Database clients created successfully",
		zap.Bool("debug", config.Debug),
		zap.Bool("cache", config.EnableCache),
	)

	return client, nil
}

// createEntClient creates a single ent client using pgx driver with optional caching
func createEntClient(ctx context.Context, dsn string, debug bool, clientType string, enableCache bool, cacheTTL time.Duration) (*ent.Client, error) {
	// Parse connection config
	connConfig, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s connection config: %w", clientType, err)
	}

	// Register connection config
	stdlib.RegisterConnConfig(connConfig)

	// Open database using pgx through stdlib interface
	db := stdlib.OpenDB(*connConfig)

	// Configure connection pool for external proxy (PgBouncer/pgpool)
	// These settings ensure connections are properly returned to proxy
	db.SetMaxOpenConns(10)                 // Maximum number of open connections to the database
	db.SetMaxIdleConns(5)                  // Maximum number of connections in the idle connection pool
	db.SetConnMaxLifetime(5 * time.Minute) // Maximum amount of time a connection may be reused
	db.SetConnMaxIdleTime(1 * time.Minute) // Maximum amount of time a connection may be idle

	// Test connection
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping %s database: %w", clientType, err)
	}

	// Create ent driver
	drv := entsql.OpenDB(dialect.Postgres, db)

	// Wrap with cache if enabled (only for query client)
	var finalDriver dialect.Driver = drv
	if enableCache && clientType == "query" {
		// Create cached driver with entcache
		cacheOpts := []entcache.Option{
			entcache.TTL(cacheTTL),
			entcache.ContextLevel(), // Context-level caching for per-request deduplication
		}

		// Attempt to add Redis cache level (no in-app LRU per project policy)
		if svc, err := redis.GetTenantCacheService(); err == nil {
			if rc := svc.GetClient(); rc != nil {
				cacheOpts = append(cacheOpts, entcache.Levels(NewTenantIsolatedRedis(rc)))
				serviceName := os.Getenv("APP_SERVICE_NAME")
				if serviceName == "" {
					serviceName = "default"
				}
				utils.Logger.Info("Redis cache level enabled for query client",
					zap.Duration("ttl", cacheTTL),
					zap.String("service", serviceName),
				)
			}
		} else {
			utils.Logger.Warn("Redis cache service unavailable, using context-level cache only",
				zap.Error(err),
			)
		}

		finalDriver = entcache.NewDriver(drv, cacheOpts...)
	}

	// Create ent client
	opts := []ent.Option{ent.Driver(finalDriver)}
	if debug {
		opts = append(opts, ent.Debug())
	}

	client := ent.NewClient(opts...)

	// Attach auto-invalidation hook on mutation client when Redis is available
	if clientType == "mutation" {
		if svc, err := redis.GetTenantCacheService(); err == nil {
			if rc := svc.GetClient(); rc != nil {
				client.Use(createAutoCacheInvalidationHook(rc))
				utils.Logger.Info("Auto-invalidation hook enabled for mutation client")
			}
		}
	}

	utils.Logger.Debug("Created database client",
		zap.String("type", clientType),
		zap.String("database", connConfig.Database),
		zap.String("host", connConfig.Host),
		zap.Uint16("port", connConfig.Port),
		zap.Bool("debug", debug),
	)

	return client, nil
}

// Query returns the query client (read-only)
func (c *Client) Query() *ent.Client {
	return c.queryClient
}

// Mutation returns the mutation client (write)
func (c *Client) Mutation() *ent.Client {
	return c.mutationClient
}

// Close closes both database connections
func (c *Client) Close() error {
	var errs []error

	if c.queryClient != nil {
		if err := c.queryClient.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close query client: %w", err))
		}
	}

	if c.mutationClient != nil {
		if err := c.mutationClient.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close mutation client: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing database clients: %v", errs)
	}

	utils.Logger.Info("Database clients closed successfully")
	return nil
}

// WithTx runs a function within a transaction on the mutation client
func (c *Client) WithTx(ctx context.Context, fn func(tx *ent.Tx) error) error {
	tx, err := c.mutationClient.Tx(ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}

	defer func() {
		if v := recover(); v != nil {
			_ = tx.Rollback()
			panic(v)
		}
	}()

	if err := fn(tx); err != nil {
		if rerr := tx.Rollback(); rerr != nil {
			err = fmt.Errorf("%w: rolling back transaction: %v", err, rerr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// EnableContextCache creates context with enabled context-level caching
// Used for GraphQL queries to avoid duplicate queries within single request
func EnableContextCache(ctx context.Context) context.Context {
	return entcache.NewContext(ctx)
}

// SkipCache creates context that skips caching (for migrations and mutations)
func SkipCache(ctx context.Context) context.Context {
	return entcache.Skip(ctx)
}

// IsDebugDB returns true if database debug mode is enabled
func IsDebugDB() bool {
	debug, _ := strconv.ParseBool(os.Getenv("DEBUG_DB"))
	return debug
}
