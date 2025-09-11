package database

import (
	"context"
	"errors"
	"fmt"
	"main/ent"
	"main/utils"
	"os"
	"time"

	"ariga.io/entcache"
	federation "github.com/esemashko/v2-federation"
	goredis "github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

const (
	// maxCacheVersion defines the maximum value for cache version before cycling back to 0
	// This prevents integer overflow and keeps cache keys shorter
	maxCacheVersion = 100000
)

var (
	// serviceName is cached after first access to avoid repeated env lookups
	serviceName string
	// redisCacheKeyPrefix includes the service name for multi-service isolation
	redisCacheKeyPrefix string
	// prefixInitialized tracks whether prefix has been initialized
	prefixInitialized bool
)

// getCacheKeyPrefix returns the cache key prefix with lazy initialization
func getCacheKeyPrefix() string {
	if !prefixInitialized {
		// Get service name from environment or use default
		serviceName = os.Getenv("APP_SERVICE_NAME")
		if serviceName == "" {
			serviceName = "default"
		}
		// Build cache prefix with service isolation
		redisCacheKeyPrefix = fmt.Sprintf("entcache:v2:service:%s:", serviceName)
		prefixInitialized = true
	}
	return redisCacheKeyPrefix
}

// tenantAwareRedisLevel implements entcache.AddGetDeleter with tenant and service isolation
type tenantAwareRedisLevel struct {
	client *goredis.Client
}

// NewTenantIsolatedRedis creates Redis cache level with tenant and service isolation
func NewTenantIsolatedRedis(client *goredis.Client) entcache.AddGetDeleter {
	return &tenantAwareRedisLevel{client: client}
}

func (t *tenantAwareRedisLevel) tenantIDFromContext(ctx context.Context) string {
	if tenantID := federation.GetTenantID(ctx); tenantID != nil {
		return tenantID.String()
	}
	return "global"
}

func (t *tenantAwareRedisLevel) versionKeyForTenant(tenantID string) string {
	return fmt.Sprintf("%stenant:%s:version", getCacheKeyPrefix(), tenantID)
}

func (t *tenantAwareRedisLevel) buildVersionedKey(ctx context.Context, key entcache.Key) (string, error) {
	tenantID := t.tenantIDFromContext(ctx)
	ver, err := t.client.Get(ctx, t.versionKeyForTenant(tenantID)).Result()
	if err != nil && !errors.Is(err, goredis.Nil) {
		return "", err
	}
	if errors.Is(err, goredis.Nil) {
		ver = "0"
	}
	return fmt.Sprintf("%stenant:%s:v%s:%v", getCacheKeyPrefix(), tenantID, ver, key), nil
}

// Add stores entry in Redis with TTL
func (t *tenantAwareRedisLevel) Add(ctx context.Context, key entcache.Key, entry *entcache.Entry, ttl time.Duration) error {
	versionedKey, err := t.buildVersionedKey(ctx, key)
	if err != nil {
		return err
	}
	data, err := entry.MarshalBinary()
	if err != nil {
		return err
	}
	if ttl <= 0 {
		// Default TTL is handled by driver-level option; fall back to 5 minutes if not set
		ttl = 5 * time.Minute
	}
	return t.client.Set(ctx, versionedKey, data, ttl).Err()
}

// Get retrieves entry from Redis
func (t *tenantAwareRedisLevel) Get(ctx context.Context, key entcache.Key) (*entcache.Entry, error) {
	versionedKey, err := t.buildVersionedKey(ctx, key)
	if err != nil {
		return nil, err
	}
	data, err := t.client.Get(ctx, versionedKey).Bytes()
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return nil, entcache.ErrNotFound
		}
		return nil, err
	}
	entry := &entcache.Entry{}
	if err := entry.UnmarshalBinary(data); err != nil {
		return nil, err
	}
	return entry, nil
}

// Del deletes entry from Redis
func (t *tenantAwareRedisLevel) Del(ctx context.Context, key entcache.Key) error {
	versionedKey, err := t.buildVersionedKey(ctx, key)
	if err != nil {
		return err
	}
	return t.client.Del(ctx, versionedKey).Err()
}

// createAutoCacheInvalidationHook increments tenant cache version in Redis on write mutations
func createAutoCacheInvalidationHook(client *goredis.Client) ent.Hook {
	return func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
			result, err := next.Mutate(ctx, m)
			if err != nil {
				return result, err
			}
			if m.Op().Is(ent.OpCreate | ent.OpUpdate | ent.OpUpdateOne | ent.OpDelete | ent.OpDeleteOne) {
				// run in background with timeout to avoid delaying response
				go func(originalCtx context.Context, mutation ent.Mutation) {
					tenantID := federation.GetTenantID(originalCtx)
					if tenantID == nil {
						return
					}
					bctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					versionKey := fmt.Sprintf("%stenant:%s:version", getCacheKeyPrefix(), tenantID.String())

					// Increment version and check if we need to cycle back to 0
					newVersion, incErr := client.Incr(bctx, versionKey).Result()
					if incErr != nil {
						utils.Logger.Error("Failed to increment cache version",
							zap.Error(incErr),
							zap.String("tenant_id", tenantID.String()),
							zap.String("entity_type", mutation.Type()),
						)
						return
					}

					// If version exceeds max, reset to 0
					// This automatically invalidates all cached entries since they use the old version
					if newVersion >= maxCacheVersion {
						if setErr := client.Set(bctx, versionKey, 0, 0).Err(); setErr != nil {
							utils.Logger.Error("Failed to reset cache version",
								zap.Error(setErr),
								zap.String("tenant_id", tenantID.String()),
								zap.Int64("version", newVersion),
							)
						} else {
							utils.Logger.Info("Cache version cycled back to 0",
								zap.String("tenant_id", tenantID.String()),
								zap.Int64("previous_version", newVersion),
							)
						}
					}
				}(ctx, m)
			}
			return result, err
		})
	}
}
