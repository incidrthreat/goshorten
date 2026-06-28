package auth

import (
	"context"
	"strconv"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type contextKey string

const (
	UserContextKey      contextKey = "auth_user"
	RoleContextKey      contextKey = "auth_role"
	SessionIDKey        contextKey = "auth_session_id"
	WorkspaceContextKey contextKey = "auth_workspace"
)

// WorkspaceHeader carries the active workspace id from the client (forwarded by
// the REST gateway into gRPC metadata).
const WorkspaceHeader = "x-workspace-id"

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
			"/Auth/OIDCAuthURL":       true, // Must be public — called before login to get redirect URL
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

		// If the token carries a JTI, verify the session is still active.
		if jti := claims.RegisteredClaims.ID; jti != "" {
			ok, err := i.AuthStore.ValidateSession(ctx, jti)
			if err != nil || !ok {
				return ctx, status.Error(codes.Unauthenticated, "session expired or revoked")
			}
		}

		// Check admin requirement
		if i.AdminMethods[method] && claims.Role != "admin" {
			return ctx, status.Error(codes.PermissionDenied, "admin access required")
		}

		ctx = context.WithValue(ctx, UserContextKey, claims.UserID)
		ctx = context.WithValue(ctx, RoleContextKey, claims.Role)
		ctx = context.WithValue(ctx, SessionIDKey, claims.RegisteredClaims.ID)

		// Resolve the active workspace: header override → session → user default.
		wsID, err := i.AuthStore.ResolveActiveWorkspace(ctx, claims.UserID,
			claims.RegisteredClaims.ID, workspaceFromMetadata(md))
		if err != nil {
			return ctx, status.Error(codes.Internal, "failed to resolve workspace")
		}
		ctx = context.WithValue(ctx, WorkspaceContextKey, wsID)
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
		// API keys are bound to a single workspace; the header cannot override it.
		ctx = context.WithValue(ctx, WorkspaceContextKey, key.WorkspaceID)
		return ctx, nil
	}

	return ctx, status.Error(codes.Unauthenticated, "invalid authorization format")
}

// workspaceFromMetadata extracts the X-Workspace-Id override from gRPC metadata.
// Returns 0 when absent or unparseable.
func workspaceFromMetadata(md metadata.MD) int64 {
	vals := md.Get(WorkspaceHeader)
	if len(vals) == 0 {
		return 0
	}
	id, err := strconv.ParseInt(strings.TrimSpace(vals[0]), 10, 64)
	if err != nil || id < 0 {
		return 0
	}
	return id
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

// SessionIDFromContext extracts the JWT ID (session ID) from context.
func SessionIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(SessionIDKey).(string)
	return id
}

// WorkspaceIDFromContext extracts the resolved active workspace id from context.
// Returns (0, false) when no workspace was resolved.
func WorkspaceIDFromContext(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(WorkspaceContextKey).(int64)
	return id, ok && id > 0
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
