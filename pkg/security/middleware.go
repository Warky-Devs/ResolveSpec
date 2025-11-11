package security

import (
	"context"
	"net/http"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	// Context keys for user information
	UserIDKey    contextKey = "user_id"
	UserRolesKey contextKey = "user_roles"
	UserTokenKey contextKey = "user_token"
)

// AuthMiddleware extracts user authentication from request and adds to context
// This should be applied before the ResolveSpec handler
// Uses GlobalSecurity.AuthenticateCallback if set, otherwise returns error
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if callback is set
		if GlobalSecurity.AuthenticateCallback == nil {
			http.Error(w, "AuthenticateCallback not set - you must provide an authentication callback", http.StatusInternalServerError)
			return
		}

		// Call the user-provided authentication callback
		userID, roles, err := GlobalSecurity.AuthenticateCallback(r)
		if err != nil {
			http.Error(w, "Authentication failed: "+err.Error(), http.StatusUnauthorized)
			return
		}

		// Add user information to context
		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		if roles != "" {
			ctx = context.WithValue(ctx, UserRolesKey, roles)
		}

		// Continue with authenticated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserID extracts the user ID from context
func GetUserID(ctx context.Context) (int, bool) {
	userID, ok := ctx.Value(UserIDKey).(int)
	return userID, ok
}

// GetUserRoles extracts user roles from context
func GetUserRoles(ctx context.Context) (string, bool) {
	roles, ok := ctx.Value(UserRolesKey).(string)
	return roles, ok
}
