package resolvers

import (
	"context"
	"main/ent"

	"github.com/google/uuid"
)

// FindUserByID returns a stub User entity for federation resolution.
// The actual User data is owned by another service (e.g., auth service).
// We only need to return an entity with the ID for GraphQL to resolve related fields.
func (r *entityResolver) FindUserByID(ctx context.Context, id uuid.UUID) (*ent.User, error) {
	// Return a stub User with only the ID set
	// This is sufficient for federation to resolve fields like user.tickets
	return &ent.User{
		ID: id,
	}, nil
}

// FindFileByID returns a File entity by its ID for federation resolution.
func (r *entityResolver) FindFileByID(ctx context.Context, id uuid.UUID) (*ent.File, error) {
	client := r.getClient(ctx)
	return client.File.Get(ctx, id)
}

// CreatedBy is the resolver for the createdBy field.
func (r *fileResolver) CreatedBy(ctx context.Context, obj *ent.File) (*ent.User, error) {
	// Return a stub User entity with the user ID from File
	// The actual User data will be resolved by the auth service in the federation
	return &ent.User{
		ID: obj.CreatedBy,
	}, nil
}
