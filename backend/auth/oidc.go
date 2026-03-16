package auth

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// OIDCProviderConfig represents a configured OIDC provider from the database.
type OIDCProviderConfig struct {
	ID            int64
	Name          string
	IssuerURL     string
	ClientID      string
	ClientSecret  string
	RedirectURI   string
	Scopes        string
	IsEnabled     bool
	AutoRegister  bool
	DefaultRole   string
}

// OIDCProviderInstance holds a live OIDC provider and its OAuth2 config.
type OIDCProviderInstance struct {
	Config   OIDCProviderConfig
	Provider *oidc.Provider
	Verifier *oidc.IDTokenVerifier
	OAuth2   *oauth2.Config
}

// OIDCManager manages multiple OIDC provider instances.
type OIDCManager struct {
	mu        sync.RWMutex
	providers map[string]*OIDCProviderInstance // keyed by provider name
}

// NewOIDCManager creates a new manager.
func NewOIDCManager() *OIDCManager {
	return &OIDCManager{
		providers: make(map[string]*OIDCProviderInstance),
	}
}

// RegisterProvider initializes and registers an OIDC provider.
func (m *OIDCManager) RegisterProvider(ctx context.Context, cfg OIDCProviderConfig) error {
	if !cfg.IsEnabled {
		return nil
	}

	provider, err := oidc.NewProvider(ctx, cfg.IssuerURL)
	if err != nil {
		return fmt.Errorf("oidc discovery for %s: %w", cfg.Name, err)
	}

	verifier := provider.Verifier(&oidc.Config{ClientID: cfg.ClientID})

	scopes := []string{oidc.ScopeOpenID}
	for _, s := range strings.Split(cfg.Scopes, " ") {
		s = strings.TrimSpace(s)
		if s != "" && s != oidc.ScopeOpenID {
			scopes = append(scopes, s)
		}
	}

	oauth2Cfg := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURI,
		Endpoint:     provider.Endpoint(),
		Scopes:       scopes,
	}

	m.mu.Lock()
	m.providers[cfg.Name] = &OIDCProviderInstance{
		Config:   cfg,
		Provider: provider,
		Verifier: verifier,
		OAuth2:   oauth2Cfg,
	}
	m.mu.Unlock()

	return nil
}

// GetProvider returns a registered provider by name.
func (m *OIDCManager) GetProvider(name string) (*OIDCProviderInstance, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.providers[name]
	return p, ok
}

// ListProviders returns the names of all registered providers.
func (m *OIDCManager) ListProviders() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	names := make([]string, 0, len(m.providers))
	for name := range m.providers {
		names = append(names, name)
	}
	return names
}

// VerifyIDToken verifies a raw ID token against the named provider.
func (m *OIDCManager) VerifyIDToken(ctx context.Context, providerName, rawToken string) (*oidc.IDToken, error) {
	p, ok := m.GetProvider(providerName)
	if !ok {
		return nil, fmt.Errorf("unknown OIDC provider: %s", providerName)
	}
	return p.Verifier.Verify(ctx, rawToken)
}

// OIDCUserInfo holds the claims extracted from an ID token.
type OIDCUserInfo struct {
	Subject  string
	Email    string
	Name     string
	Issuer   string
}

// ExtractUserInfo extracts standard claims from an ID token.
func ExtractUserInfo(token *oidc.IDToken) (*OIDCUserInfo, error) {
	var claims struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := token.Claims(&claims); err != nil {
		return nil, fmt.Errorf("extract claims: %w", err)
	}
	return &OIDCUserInfo{
		Subject: token.Subject,
		Email:   claims.Email,
		Name:    claims.Name,
		Issuer:  token.Issuer,
	}, nil
}
