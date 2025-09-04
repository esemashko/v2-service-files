package dataloader

import (
	"context"
	"main/ent"

	"github.com/google/uuid"
)

// contextKey for preload cache
type preloadCacheKey struct{}

// PreloadCache stores pre-loaded entities to avoid duplicate queries
type PreloadCache struct {
	Tenants map[uuid.UUID]*ent.Tenant // user ID -> user
}

// GetPreloadCache retrieves the preload cache from context
func GetPreloadCache(ctx context.Context) *PreloadCache {
	cache, _ := ctx.Value(preloadCacheKey{}).(*PreloadCache)
	if cache == nil {
		cache = &PreloadCache{
			Tenants: make(map[uuid.UUID]*ent.Tenant),
		}
	}
	return cache
}

// WithPreloadCache adds a preload cache to the context
func WithPreloadCache(ctx context.Context, cache *PreloadCache) context.Context {
	return context.WithValue(ctx, preloadCacheKey{}, cache)
}

// PopulateFromTenants populates the cache from loaded users
func (c *PreloadCache) PopulateFromTenants(users []*ent.Tenant) {
	for _, user := range users {
		c.Tenants[user.ID] = user
	}
}
