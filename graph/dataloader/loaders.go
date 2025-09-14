package dataloader

import (
	"context"
	"main/ent"
	"time"

	"github.com/google/uuid"
)

type ctxKey string

const (
	LoadersKey = ctxKey("dataloaders")
)

// Loaders holds all data loaders
type Loaders struct {
	// Federation entity loaders - for resolving entities from other services
	//FederationTenantLoader *BatchLoader[uuid.UUID, *ent.Tenant]

	// File permission loaders
	FileCanDeleteLoader *BatchLoader[uuid.UUID, bool]
}

// NewLoaders creates new data loaders
func NewLoaders(client *ent.Client) *Loaders {
	// Federation readers
	//federationTenantReader := NewFederationTenantReader(client)

	// File permission readers
	fileDeletePermissionReader := NewFileDeletePermissionReader(client)

	return &Loaders{
		// Federation loaders
		//FederationTenantLoader: NewBatchLoader(federationTenantReader.GetTenantsByID, 2*time.Millisecond, 100),

		// File permission loaders
		FileCanDeleteLoader: NewBatchLoader(fileDeletePermissionReader.GetCanDeleteFlags, 2*time.Millisecond, 100),
	}
}

// For returns the loaders from context
func For(ctx context.Context) *Loaders {
	return ctx.Value(LoadersKey).(*Loaders)
}

// WithLoaders stores the loaders in the context
func WithLoaders(ctx context.Context, loaders *Loaders) context.Context {
	return context.WithValue(ctx, LoadersKey, loaders)
}

// GetFileCanDelete returns canDelete flag for a single file
func GetFileCanDelete(ctx context.Context, fileID uuid.UUID) (bool, error) {
	loaders := For(ctx)
	return loaders.FileCanDeleteLoader.Load(ctx, fileID)
}

// GetFederationTenant gets a Tenant entity for federation resolution
// This is used by entity resolvers when other services request Tenant entities
/*func GetFederationTenant(ctx context.Context, userID uuid.UUID) (*ent.Tenant, error) {
	loaders := For(ctx)
	if loaders == nil {
		// Если DataLoader не инициализирован, получаем клиент из контекста и создаем loader
		client := ent.FromContext(ctx)
		if client == nil {
			return nil, nil
		}
		// Создаем временный loader для этого запроса
		loaders = NewLoaders(client)
		ctx = context.WithValue(ctx, LoadersKey, loaders)
	}
	return loaders.FederationTenantLoader.Load(ctx, userID)
}*/
