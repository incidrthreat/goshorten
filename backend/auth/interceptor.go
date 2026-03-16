package auth

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type contextKey string

const (
	UserContextKey contextKey = "auth_user"
	RoleContextKey contextKey = "auth_role"
)

// AuthInterceptor validates authentication on gRPC calls.
type AuthInterceptor struct {
	JWTMgr    *JWTManager
	AuthStore *AuthStore
	// Methods that don't require authentication (e.g., redirect lookups).
	PublicMethods map[string]bool
	// Methods that require admin role.
	AdminMethods map[string]bool
}

// NewAuthInterceptor creates an interceptor with default public/admin method lists.
func NewAuthInterceptor(jwtMgr *JWTManager, store *AuthStore) *AuthInterceptor {
	return &AuthInterceptor{
		JWTMgr:    jwtMgr,
		AuthStore: store,
		PublicMethods: map[string]bool{
			"/Shortener/GetURL":      true, // Redirects must be unauthenticated
			"/Shortener/PreviewURL":  true, // Public link preview (code+)
			"/Auth/Login":             true,
			"/Auth/OIDCCallback":      true,
			"/Auth/ListOIDCProviders": true,
		},
		AdminMethods: map[string]bool{
			"/Auth/CreateOIDCProvider":    true,
			"/Auth/DeleteOIDCProvider":    true,
			"/Shortener/GetOrphanVisits":  true,
		},
	}
}

// Unary returns a gRPC unary server interceptor.
func (i *AuthInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		ctx, err := i.authorize(ctx, info.FullMethod)
		if err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

func (i *AuthInterceptor) authorize(ctx context.Context, method string) (context.Context, error) {
	// Public methods skip auth
	if i.PublicMethods[method] {
		return ctx, nil
	}

	// Extract token from metadata
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx, status.Error(codes.Unauthenticated, "missing metadata")
	}

	values := md.Get("authorization")
	if len(values) == 0 {
		return ctx, status.Error(codes.Unauthenticated, "missing authorization header")
	}

	token := values[0]

	// Try Bearer JWT token first
	if strings.HasPrefix(token, "Bearer ") {
		jwtToken := strings.TrimPrefix(token, "Bearer ")
		claims, err := i.JWTMgr.Verify(jwtToken)
		if err != nil {
			return ctx, status.Error(codes.Unauthenticated, "invalid token")
		}

		// Check admin requirement
		if i.AdminMethods[method] && claims.Role != "admin" {
			return ctx, status.Error(codes.PermissionDenied, "admin access required")
		}

		ctx = context.WithValue(ctx, UserContextKey, claims.UserID)
		ctx = context.WithValue(ctx, RoleContextKey, claims.Role)
		return ctx, nil
	}

	// Try API key (sent as "ApiKey <key>")
	if strings.HasPrefix(token, "ApiKey ") {
		apiKey := strings.TrimPrefix(token, "ApiKey ")
		user, key, err := i.AuthStore.ValidateAPIKey(ctx, apiKey)
		if err != nil {
			return ctx, status.Error(codes.Unauthenticated, "invalid API key")
		}

		// Check admin requirement
		if i.AdminMethods[method] && user.Role != "admin" {
			return ctx, status.Error(codes.PermissionDenied, "admin access required")
		}

		// Check scope
		if !hasScope(key.Scopes, methodToScope(method)) {
			return ctx, status.Error(codes.PermissionDenied, "insufficient scope")
		}

		ctx = context.WithValue(ctx, UserContextKey, user.ID)
		ctx = context.WithValue(ctx, RoleContextKey, user.Role)
		return ctx, nil
	}

	return ctx, status.Error(codes.Unauthenticated, "invalid authorization format")
}

// UserIDFromContext extracts the authenticated user ID from context.
func UserIDFromContext(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(UserContextKey).(int64)
	return id, ok
}

// RoleFromContext extracts the authenticated user role from context.
func RoleFromContext(ctx context.Context) string {
	role, _ := ctx.Value(RoleContextKey).(string)
	return role
}

func hasScope(keyScopes, required string) bool {
	if required == "" {
		return true
	}
	for _, s := range strings.Split(keyScopes, ",") {
		if strings.TrimSpace(s) == required {
			return true
		}
	}
	return false
}

func methodToScope(method string) string {
	switch {
	case strings.Contains(method, "Create"), strings.Contains(method, "Update"), strings.Contains(method, "Delete"):
		return "urls:write"
	case strings.Contains(method, "Get"), strings.Contains(method, "List"), strings.Contains(method, "Stats"),
		strings.Contains(method, "Visit"):
		return "urls:read"
	case strings.Contains(method, "APIKey"):
		return "keys:manage"
	default:
		return ""
	}
}
