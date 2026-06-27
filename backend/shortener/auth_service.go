package shortener

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/incidrthreat/goshorten/backend/auth"
	pb "github.com/incidrthreat/goshorten/backend/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// AuthServer implements the Auth gRPC service.
type AuthServer struct {
	pb.UnimplementedAuthServer
	AuthStore *auth.AuthStore
	JWTMgr    *auth.JWTManager
	OIDCMgr   *auth.OIDCManager
}

// Login authenticates with email/password (break-glass admin or local accounts).
func (s *AuthServer) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	if req.GetEmail() == "" || req.GetPassword() == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password required")
	}

	user, err := s.AuthStore.GetUserByEmail(ctx, req.GetEmail())
	if err != nil {
		return nil, status.Error(codes.Internal, "authentication failed")
	}
	if user == nil || user.PasswordHash == nil {
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}
	if !user.IsActive {
		return nil, status.Error(codes.PermissionDenied, "account disabled")
	}

	if !auth.CheckPassword(req.GetPassword(), *user.PasswordHash) {
		s.AuthStore.LogSignIn(ctx, &user.ID, clientIP(ctx), clientUA(ctx), false)
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}

	token, jti, err := s.JWTMgr.Generate(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate token")
	}

	expiry := time.Now().Add(time.Duration(s.JWTMgr.ExpiryHr) * time.Hour)
	ip, ua := clientIP(ctx), clientUA(ctx)
	_ = s.AuthStore.CreateSession(ctx, jti, user.ID, ip, ua, sessionLabel(ua), expiry)
	s.AuthStore.LogSignIn(ctx, &user.ID, ip, ua, true)

	return &pb.LoginResponse{
		Token: token,
		User:  userToProto(user),
	}, nil
}

// OIDCAuthURL returns the authorization URL for a given OIDC provider.
func (s *AuthServer) OIDCAuthURL(ctx context.Context, req *pb.OIDCAuthURLRequest) (*pb.OIDCAuthURLResponse, error) {
	provider, ok := s.OIDCMgr.GetProvider(req.GetProviderName())
	if !ok {
		return nil, status.Error(codes.NotFound, "OIDC provider not found")
	}

	state, err := generateState()
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate state")
	}

	url := provider.OAuth2.AuthCodeURL(state)
	return &pb.OIDCAuthURLResponse{
		AuthUrl: url,
		State:   state,
	}, nil
}

// OIDCCallback handles the OIDC callback after user authenticates with the IdP.
func (s *AuthServer) OIDCCallback(ctx context.Context, req *pb.OIDCCallbackRequest) (*pb.LoginResponse, error) {
	provider, ok := s.OIDCMgr.GetProvider(req.GetProviderName())
	if !ok {
		return nil, status.Error(codes.NotFound, "OIDC provider not found")
	}

	// Exchange authorization code for tokens
	oauth2Token, err := provider.OAuth2.Exchange(ctx, req.GetCode())
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "failed to exchange code")
	}

	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		return nil, status.Error(codes.Internal, "missing id_token")
	}

	// Verify the ID token
	idToken, err := provider.Verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid id_token")
	}

	// Extract user info from claims
	userInfo, err := auth.ExtractUserInfo(idToken)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to extract user info")
	}

	// Look up or auto-register user
	user, err := s.AuthStore.GetUserByOIDC(ctx, userInfo.Issuer, userInfo.Subject)
	if err != nil {
		return nil, status.Error(codes.Internal, "database error")
	}

	if user == nil {
		if !provider.Config.AutoRegister {
			return nil, status.Error(codes.PermissionDenied, "auto-registration disabled for this provider")
		}

		email := userInfo.Email
		if email == "" {
			return nil, status.Error(codes.InvalidArgument, "email claim required but not provided by IdP")
		}

		user, err = s.AuthStore.CreateUser(ctx, email, userInfo.Name,
			provider.Config.DefaultRole, nil, &userInfo.Issuer, &userInfo.Subject)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to create user")
		}
	}

	if !user.IsActive {
		return nil, status.Error(codes.PermissionDenied, "account disabled")
	}

	token, jti, err := s.JWTMgr.Generate(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate token")
	}

	expiry := time.Now().Add(time.Duration(s.JWTMgr.ExpiryHr) * time.Hour)
	ip, ua := clientIP(ctx), clientUA(ctx)
	_ = s.AuthStore.CreateSession(ctx, jti, user.ID, ip, ua, sessionLabel(ua), expiry)
	s.AuthStore.LogSignIn(ctx, &user.ID, ip, ua, true)

	return &pb.LoginResponse{
		Token: token,
		User:  userToProto(user),
	}, nil
}

// ListOIDCProviders returns the names of available OIDC providers (public).
func (s *AuthServer) ListOIDCProviders(ctx context.Context, req *pb.ListOIDCProvidersRequest) (*pb.ListOIDCProvidersResponse, error) {
	configs, err := s.AuthStore.ListOIDCProviders(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list providers")
	}

	resp := &pb.ListOIDCProvidersResponse{}
	for _, cfg := range configs {
		if cfg.IsEnabled {
			resp.Providers = append(resp.Providers, &pb.OIDCProvider{
				Id:           cfg.ID,
				Name:         cfg.Name,
				IssuerUrl:    cfg.IssuerURL,
				IsEnabled:    cfg.IsEnabled,
				AutoRegister: cfg.AutoRegister,
				DefaultRole:  cfg.DefaultRole,
			})
		}
	}
	return resp, nil
}

// CreateOIDCProvider adds a new OIDC provider (admin only).
func (s *AuthServer) CreateOIDCProvider(ctx context.Context, req *pb.CreateOIDCProviderRequest) (*pb.OIDCProvider, error) {
	if req.GetName() == "" || req.GetIssuerUrl() == "" || req.GetClientId() == "" {
		return nil, status.Error(codes.InvalidArgument, "name, issuer_url, and client_id required")
	}

	cfg := auth.OIDCProviderConfig{
		Name:         req.GetName(),
		IssuerURL:    req.GetIssuerUrl(),
		ClientID:     req.GetClientId(),
		ClientSecret: req.GetClientSecret(),
		RedirectURI:  req.GetRedirectUri(),
		Scopes:       req.GetScopes(),
		IsEnabled:    true,
		AutoRegister: req.GetAutoRegister(),
		DefaultRole:  req.GetDefaultRole(),
	}
	if cfg.Scopes == "" {
		cfg.Scopes = "openid email profile"
	}
	if cfg.DefaultRole == "" {
		cfg.DefaultRole = "user"
	}

	saved, err := s.AuthStore.CreateOIDCProvider(ctx, cfg)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to create provider")
	}

	// Register it live
	if err := s.OIDCMgr.RegisterProvider(ctx, *saved); err != nil {
		log.Warn("OIDC", "Failed to register provider live (will work after restart)", err)
	}

	return &pb.OIDCProvider{
		Id:           saved.ID,
		Name:         saved.Name,
		IssuerUrl:    saved.IssuerURL,
		IsEnabled:    saved.IsEnabled,
		AutoRegister: saved.AutoRegister,
		DefaultRole:  saved.DefaultRole,
	}, nil
}

// DeleteOIDCProvider removes an OIDC provider (admin only).
func (s *AuthServer) DeleteOIDCProvider(ctx context.Context, req *pb.DeleteOIDCProviderRequest) (*pb.DeleteOIDCProviderResponse, error) {
	if req.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "provider name required")
	}

	if err := s.AuthStore.DeleteOIDCProvider(ctx, req.GetName()); err != nil {
		return nil, status.Error(codes.Internal, "failed to delete provider")
	}

	return &pb.DeleteOIDCProviderResponse{Success: true}, nil
}

// CreateAPIKey generates a new API key for the authenticated user.
func (s *AuthServer) CreateAPIKey(ctx context.Context, req *pb.CreateAPIKeyRequest) (*pb.CreateAPIKeyResponse, error) {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "not authenticated")
	}

	plaintext, key, err := s.AuthStore.CreateAPIKey(ctx, userID, req.GetLabel(), req.GetScopes())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to create API key")
	}

	return &pb.CreateAPIKeyResponse{
		PlaintextKey: plaintext,
		Key:          apiKeyToProto(key, plaintext[:8]),
	}, nil
}

// ListAPIKeys returns all API keys for the authenticated user.
func (s *AuthServer) ListAPIKeys(ctx context.Context, req *pb.ListAPIKeysRequest) (*pb.ListAPIKeysResponse, error) {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "not authenticated")
	}

	keys, err := s.AuthStore.ListAPIKeys(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list API keys")
	}

	resp := &pb.ListAPIKeysResponse{}
	for _, k := range keys {
		// We don't have the plaintext, so prefix is just "****"
		resp.Keys = append(resp.Keys, apiKeyToProto(&k, "********"))
	}
	return resp, nil
}

// RevokeAPIKey revokes an API key owned by the authenticated user.
func (s *AuthServer) RevokeAPIKey(ctx context.Context, req *pb.RevokeAPIKeyRequest) (*pb.RevokeAPIKeyResponse, error) {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "not authenticated")
	}

	if err := s.AuthStore.RevokeAPIKey(ctx, req.GetKeyId(), userID); err != nil {
		if errors.Is(err, errors.New("API key not found")) {
			return nil, status.Error(codes.NotFound, "API key not found")
		}
		return nil, status.Error(codes.Internal, "failed to revoke API key")
	}

	return &pb.RevokeAPIKeyResponse{Success: true}, nil
}

// RollAPIKey revokes an existing key and creates a new one with the same label/scopes.
func (s *AuthServer) RollAPIKey(ctx context.Context, req *pb.RollAPIKeyRequest) (*pb.CreateAPIKeyResponse, error) {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "not authenticated")
	}

	plaintext, key, err := s.AuthStore.RollAPIKey(ctx, req.GetKeyId(), userID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.CreateAPIKeyResponse{
		PlaintextKey: plaintext,
		Key:          apiKeyToProto(key, plaintext[:8]),
	}, nil
}

// GetCurrentUser returns info about the authenticated user.
func (s *AuthServer) GetCurrentUser(ctx context.Context, req *pb.GetCurrentUserRequest) (*pb.UserInfo, error) {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "not authenticated")
	}

	user, err := s.AuthStore.GetUserByID(ctx, userID)
	if err != nil || user == nil {
		return nil, status.Error(codes.Internal, "failed to get user")
	}

	return userToProto(user), nil
}

// --- Helpers ---

func userToProto(u *auth.User) *pb.UserInfo {
	return &pb.UserInfo{
		Id:        u.ID,
		Email:     u.Email,
		Name:      u.Name,
		Role:      u.Role,
		CreatedAt: timestamppb.New(u.CreatedAt),
	}
}

func apiKeyToProto(k *auth.APIKey, prefix string) *pb.APIKeyInfo {
	info := &pb.APIKeyInfo{
		Id:        k.ID,
		Label:     k.Label,
		Scopes:    k.Scopes,
		CreatedAt: timestamppb.New(k.CreatedAt),
		Revoked:   k.Revoked,
		KeyPrefix: prefix,
	}
	if k.ExpiresAt != nil {
		info.ExpiresAt = timestamppb.New(*k.ExpiresAt)
	}
	return info
}

func generateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// clientIP extracts the client IP from gRPC metadata (forwarded by the gateway).
func clientIP(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	for _, key := range []string{"x-real-ip", "x-forwarded-for"} {
		if vals := md.Get(key); len(vals) > 0 {
			// x-forwarded-for may be a comma-separated list; take the first
			return strings.TrimSpace(strings.SplitN(vals[0], ",", 2)[0])
		}
	}
	return ""
}

// clientUA extracts the User-Agent from gRPC metadata.
func clientUA(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	if vals := md.Get("user-agent"); len(vals) > 0 {
		return vals[0]
	}
	return ""
}

// sessionLabel derives a short human-readable label from a user-agent string.
func sessionLabel(ua string) string {
	ua = strings.ToLower(ua)
	browser := "Unknown browser"
	os := "Unknown OS"

	switch {
	case strings.Contains(ua, "edg/"):
		browser = "Edge"
	case strings.Contains(ua, "chrome"):
		browser = "Chrome"
	case strings.Contains(ua, "firefox"):
		browser = "Firefox"
	case strings.Contains(ua, "safari"):
		browser = "Safari"
	case strings.Contains(ua, "curl"):
		browser = "curl"
	}

	switch {
	case strings.Contains(ua, "windows"):
		os = "Windows"
	case strings.Contains(ua, "mac os"):
		os = "macOS"
	case strings.Contains(ua, "linux"):
		os = "Linux"
	case strings.Contains(ua, "android"):
		os = "Android"
	case strings.Contains(ua, "ios") || strings.Contains(ua, "iphone"):
		os = "iOS"
	}

	return browser + " on " + os
}
