package privacy

// Privacy error constants
const (
	// Authentication errors
	ErrAuthenticationRequired = "authentication required"
	ErrUserNotFound           = "user not found"
	ErrUserInactive           = "user account is inactive"

	// Authorization errors
	ErrInsufficientPermissions = "insufficient permissions"
	ErrAccessDenied            = "access denied"
	ErrNotDepartmentHead       = "not department head"

	// System errors
	ErrInvalidContext = "invalid context"
	ErrInvalidEntity  = "invalid entity"

	// Field update errors
	ErrFieldNotAllowed = "only allowed fields can be modified"
)
