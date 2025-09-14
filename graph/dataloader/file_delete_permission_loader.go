package dataloader

import (
	"context"
	"main/ent"
	"main/ent/file"

	federation "github.com/esemashko/v2-federation"

	"github.com/google/uuid"
)

// FileDeletePermissionReader batches canDelete checks for File entities
type FileDeletePermissionReader struct {
	client *ent.Client
}

func NewFileDeletePermissionReader(client *ent.Client) *FileDeletePermissionReader {
	return &FileDeletePermissionReader{client: client}
}

// GetCanDeleteFlags returns canDelete flags for the given file IDs preserving input order
func (r *FileDeletePermissionReader) GetCanDeleteFlags(ctx context.Context, fileIDs []uuid.UUID) ([]bool, []error) {
	results := make([]bool, len(fileIDs))
	errors := make([]error, len(fileIDs))

	if len(fileIDs) == 0 {
		return results, errors
	}

	// Get current user from federation context
	userID := federation.GetUserID(ctx)
	if userID == nil {
		// No user in context - can't delete
		for i := range results {
			results[i] = false
		}
		return results, errors
	}

	userRole := federation.GetUserRole(ctx)

	// Admin can delete any file
	if userRole == "admin" || userRole == "owner" {
		for i := range results {
			results[i] = true
		}
		return results, errors
	}

	// Wrap context with client for hooks/privacies per project rules
	ctxWithClient := ent.NewContext(ctx, r.client)

	// Find files created by the current user among requested IDs
	ownedIDs, err := r.client.File.Query().
		Where(
			file.IDIn(fileIDs...),
			file.CreatedBy(*userID),
		).
		IDs(ctxWithClient)
	if err != nil {
		for i := range errors {
			errors[i] = err
		}
		return results, errors
	}

	ownedSet := make(map[uuid.UUID]struct{}, len(ownedIDs))
	for _, id := range ownedIDs {
		ownedSet[id] = struct{}{}
	}

	for i, id := range fileIDs {
		_, ok := ownedSet[id]
		results[i] = ok
	}

	return results, errors
}
