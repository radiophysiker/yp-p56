package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/radiophysiker/d56/internal/domain/user"
	"github.com/radiophysiker/d56/internal/infrastructure/jwt"
)

type contextKey string

// UserIDKey is the key used to store UserID in context
const UserIDKey contextKey = "user_id"

// AuthMiddleware verifies the JWT token and adds the UserID to the context
type AuthMiddleware struct {
	jwtService *jwt.Service
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(jwtService *jwt.Service) *AuthMiddleware {
	return &AuthMiddleware{
		jwtService: jwtService,
	}
}

// RequireAuth verifies the JWT token and adds the UserID to the context
func (m *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header is required", http.StatusUnauthorized)
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := m.jwtService.ParseToken(tokenString)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserIDFromContext extracts the UserID from the context
// This function is located in the Infrastructure layer as it is aware of the HTTP context
func GetUserIDFromContext(ctx context.Context) (user.UserID, bool) {
	userID, ok := ctx.Value(UserIDKey).(user.UserID)
	return userID, ok
}
