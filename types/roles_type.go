package types

// Константы для ролей пользователей
const (
	// RoleOwner - владелец организации
	RoleOwner = "owner"
	// RoleAdmin - администратор организации
	RoleAdmin = "admin"
	// RoleMember - обычный пользователь организации
	RoleMember = "member"
	// RoleClient - клиент организации
	RoleClient = "client"
)

// IsRoleHigherOrEqual проверяет, является ли роль1 выше или равной роли2
func IsRoleHigherOrEqual(role1, role2 string) bool {
	roleHierarchy := map[string]int{
		RoleClient: 1,
		RoleMember: 2,
		RoleAdmin:  3,
		RoleOwner:  4,
	}

	level1, exists1 := roleHierarchy[role1]
	level2, exists2 := roleHierarchy[role2]

	// Если одна из ролей не существует, возвращаем false
	if !exists1 || !exists2 {
		return false
	}

	return level1 >= level2
}
