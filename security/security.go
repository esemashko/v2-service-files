package security

import (
	"context"
	"errors"

	federation "github.com/esemashko/v2-federation"
)

// ValidateAuthAccess проверяет базовую авторизацию пользователя по заголовку
func ValidateAuthAccess(ctx context.Context) error {
	userID := federation.GetUserID(ctx)
	if userID == nil {
		return errors.New("you are not authenticated")
	}

	tenantID := federation.GetTenantID(ctx)
	if tenantID == nil {
		return errors.New("you are not authenticated")
	}

	return nil
}
