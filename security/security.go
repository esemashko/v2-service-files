package security

import (
	"context"
	"errors"
	"main/types"

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

// ValidateAdminAccess проверяет пользователя на администратора
func ValidateAdminAccess(ctx context.Context) error {
	err := ValidateAuthAccess(ctx)
	if err != nil {
		return err
	}

	userRole := federation.GetUserRole(ctx)
	if userRole == "" {
		return errors.New("you are not authenticated")
	}

	if types.IsRoleHigherOrEqual(userRole, types.RoleAdmin) {
		return nil
	}

	return errors.New("you are not authenticated")
}

// ValidateMemberAccess проверяет пользователя на роль сотрудника
func ValidateMemberAccess(ctx context.Context) error {
	err := ValidateAuthAccess(ctx)
	if err != nil {
		return err
	}

	userRole := federation.GetUserRole(ctx)
	if userRole == "" {
		return errors.New("you are not authenticated")
	}

	if types.IsRoleHigherOrEqual(userRole, types.RoleMember) {
		return nil
	}

	return errors.New("you are not authenticated")
}
