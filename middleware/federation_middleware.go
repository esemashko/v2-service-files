package middleware

import (
	"net/http"

	federation "github.com/esemashko/v2-federation"
)

// FederationMiddleware applies federation middleware and logs federation context
func FederationMiddleware(next http.Handler) http.Handler {
	// First apply federation middleware
	handler := federation.Middleware(next)

	// Then add logging
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get context from request
		ctx := r.Context()

		// Log federation context if present
		if fedCtx := federation.GetContext(ctx); fedCtx != nil {
			/*utils.Logger.Debug("Federation context",
				zap.String("requestID", fedCtx.RequestID),
				zap.Any("tenantID", fedCtx.TenantID),
				zap.Any("userID", fedCtx.UserID),
				zap.Any("sessionID", fedCtx.SessionID),
				zap.String("userRole", fedCtx.UserRole),
				zap.String("language", fedCtx.Language),
				zap.Any("departmentIDs", fedCtx.DepartmentIDs),
				zap.Any("managedDepartmentIDs", fedCtx.ManagedDepartmentIDs),
				zap.String("deviceID", fedCtx.DeviceID),
				zap.String("fingerprint", fedCtx.Fingerprint),
				zap.Strings("scopes", fedCtx.Scopes),
				zap.String("userAgent", fedCtx.UserAgent),
				zap.String("clientIP", fedCtx.ClientIP),
				zap.String("forwardedHost", fedCtx.ForwardedHost),
			)*/
		}

		// Call the federation-wrapped handler
		handler.ServeHTTP(w, r)
	})
}
