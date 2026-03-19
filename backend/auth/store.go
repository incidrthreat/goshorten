package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var log = hclog.Default()

// User represents a user record.
type User struct {
	ID           int64
	Email        string
	Name         string
	Role         string
	PasswordHash *string
	OIDCSubject  *string
	OIDCIssuer   *string
	IsActive     bool
	CreatedAt    time.Time
}

// APIKey represents an API key record.
type APIKey struct {
	ID        int64
	UserID    int64
	KeyHash   string
	Label     string
	Scopes    string
	CreatedAt time.Time
	ExpiresAt *time.Time
	Revoked   bool
}

// AuthStore handles auth-related database operations.
type AuthStore struct {
	Pool *pgxpool.Pool
}

// NewAuthStore creates a new auth store.
func NewAuthStore(pool *pgxpool.Pool) *AuthStore {
	return &AuthStore{Pool: pool}
}

// --- User operations ---

// GetUserByEmail looks up a user by email.
func (s *AuthStore) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	u := &User{}
	err := s.Pool.QueryRow(ctx,
		`SELECT id, email, COALESCE(name, ''), role, password_hash, oidc_subject, oidc_issuer, is_active, created_at
		 FROM users WHERE email = $1`, email,
	).Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.PasswordHash, &u.OIDCSubject, &u.OIDCIssuer, &u.IsActive, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return u, nil
}

// GetUserByID looks up a user by ID.
func (s *AuthStore) GetUserByID(ctx context.Context, id int64) (*User, error) {
	u := &User{}
	err := s.Pool.QueryRow(ctx,
		`SELECT id, email, COALESCE(name, ''), role, password_hash, oidc_subject, oidc_issuer, is_active, created_at
		 FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.PasswordHash, &u.OIDCSubject, &u.OIDCIssuer, &u.IsActive, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return u, nil
}

// GetUserByOIDC looks up a user by OIDC issuer + subject.
func (s *AuthStore) GetUserByOIDC(ctx context.Context, issuer, subject string) (*User, error) {
	u := &User{}
	err := s.Pool.QueryRow(ctx,
		`SELECT id, email, COALESCE(name, ''), role, password_hash, oidc_subject, oidc_issuer, is_active, created_at
		 FROM users WHERE oidc_issuer = $1 AND oidc_subject = $2`, issuer, subject,
	).Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.PasswordHash, &u.OIDCSubject, &u.OIDCIssuer, &u.IsActive, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get user by oidc: %w", err)
	}
	return u, nil
}

// CreateUser inserts a new user.
func (s *AuthStore) CreateUser(ctx context.Context, email, name, role string, passwordHash *string, oidcIssuer, oidcSubject *string) (*User, error) {
	u := &User{}
	err := s.Pool.QueryRow(ctx,
		`INSERT INTO users (email, name, role, password_hash, oidc_issuer, oidc_subject)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, email, COALESCE(name, ''), role, password_hash, oidc_subject, oidc_issuer, is_active, created_at`,
		email, name, role, passwordHash, oidcIssuer, oidcSubject,
	).Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.PasswordHash, &u.OIDCSubject, &u.OIDCIssuer, &u.IsActive, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}

// DeleteUser permanently removes a user record.
func (s *AuthStore) DeleteUser(ctx context.Context, userID int64) error {
	result, err := s.Pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	if result.RowsAffected() == 0 {
		return errors.New("user not found")
	}
	return nil
}

// UpdateUserPassword updates a user's password hash.
func (s *AuthStore) UpdateUserPassword(ctx context.Context, userID int64, hash string) error {
	_, err := s.Pool.Exec(ctx,
		`UPDATE users SET password_hash = $1 WHERE id = $2`, hash, userID)
	return err
}

// UpdateUserEmail updates a user's email address.
func (s *AuthStore) UpdateUserEmail(ctx context.Context, userID int64, email string) error {
	_, err := s.Pool.Exec(ctx,
		`UPDATE users SET email = $1 WHERE id = $2`, email, userID)
	return err
}

// UpdateUserParams holds fields that an admin can change on a user.
type UpdateUserParams struct {
	Role     *string
	IsActive *bool
	Email    *string
	Name     *string
}

// UpdateUser applies admin-level changes to a user record.
func (s *AuthStore) UpdateUser(ctx context.Context, userID int64, params UpdateUserParams) (*User, error) {
	setClauses := []string{}
	args := []interface{}{}
	idx := 1

	if params.Role != nil {
		setClauses = append(setClauses, fmt.Sprintf("role = $%d", idx))
		args = append(args, *params.Role)
		idx++
	}
	if params.IsActive != nil {
		setClauses = append(setClauses, fmt.Sprintf("is_active = $%d", idx))
		args = append(args, *params.IsActive)
		idx++
	}
	if params.Email != nil {
		setClauses = append(setClauses, fmt.Sprintf("email = $%d", idx))
		args = append(args, *params.Email)
		idx++
	}
	if params.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", idx))
		args = append(args, *params.Name)
		idx++
	}

	if len(setClauses) == 0 {
		return s.GetUserByID(ctx, userID)
	}

	query := fmt.Sprintf("UPDATE users SET %s WHERE id = $%d",
		strings.Join(setClauses, ", "), idx)
	args = append(args, userID)

	if _, err := s.Pool.Exec(ctx, query, args...); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}
	return s.GetUserByID(ctx, userID)
}

// ListUsers returns a paginated list of users with optional search.
func (s *AuthStore) ListUsers(ctx context.Context, search string, page, pageSize int) ([]User, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	args := []interface{}{}
	idx := 1
	whereSQL := ""
	if search != "" {
		whereSQL = fmt.Sprintf("WHERE email ILIKE $%d OR COALESCE(name, '') ILIKE $%d", idx, idx)
		args = append(args, "%"+search+"%")
		idx++
	}

	var total int
	if err := s.Pool.QueryRow(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM users %s", whereSQL), args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	offset := (page - 1) * pageSize
	rows, err := s.Pool.Query(ctx,
		fmt.Sprintf(`SELECT id, email, COALESCE(name, ''), role, password_hash, oidc_subject, oidc_issuer, is_active, created_at
		 FROM users %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
			whereSQL, idx, idx+1),
		append(args, pageSize, offset)...)
	if err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.PasswordHash,
			&u.OIDCSubject, &u.OIDCIssuer, &u.IsActive, &u.CreatedAt); err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}
	return users, total, nil
}

// --- Break-glass admin bootstrap ---

// BootstrapAdmin creates the initial admin account if it doesn't exist.
// Returns true if the admin was created, false if it already existed.
func (s *AuthStore) BootstrapAdmin(ctx context.Context, email, password string) (bool, error) {
	if email == "" || password == "" {
		return false, nil
	}

	existing, err := s.GetUserByEmail(ctx, email)
	if err != nil {
		return false, err
	}
	if existing != nil {
		// Update password hash in case it changed in config
		hash, err := HashPassword(password)
		if err != nil {
			return false, err
		}
		if err := s.UpdateUserPassword(ctx, existing.ID, hash); err != nil {
			return false, err
		}
		log.Info("Auth", "Break-glass admin", "password synced", "email", email)
		return false, nil
	}

	hash, err := HashPassword(password)
	if err != nil {
		return false, err
	}

	_, err = s.CreateUser(ctx, email, "Admin", "admin", &hash, nil, nil)
	if err != nil {
		return false, fmt.Errorf("bootstrap admin: %w", err)
	}

	log.Info("Auth", "Break-glass admin created", email)
	return true, nil
}

// --- API Key operations ---

// CreateAPIKey creates a new API key for a user.
// Returns (plaintext_key, *APIKey, error). The plaintext is shown once and never stored.
func (s *AuthStore) CreateAPIKey(ctx context.Context, userID int64, label, scopes string) (string, *APIKey, error) {
	plaintext, hash, err := GenerateAPIKey()
	if err != nil {
		return "", nil, err
	}

	if scopes == "" {
		scopes = "urls:read,urls:write"
	}

	k := &APIKey{}
	err = s.Pool.QueryRow(ctx,
		`INSERT INTO api_keys (user_id, key_hash, label, scopes)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, user_id, key_hash, label, scopes, created_at, expires_at, revoked`,
		userID, hash, label, scopes,
	).Scan(&k.ID, &k.UserID, &k.KeyHash, &k.Label, &k.Scopes, &k.CreatedAt, &k.ExpiresAt, &k.Revoked)
	if err != nil {
		return "", nil, fmt.Errorf("create api key: %w", err)
	}

	return plaintext, k, nil
}

// ValidateAPIKey looks up a user by API key plaintext.
func (s *AuthStore) ValidateAPIKey(ctx context.Context, plaintext string) (*User, *APIKey, error) {
	hash := HashAPIKey(plaintext)

	k := &APIKey{}
	err := s.Pool.QueryRow(ctx,
		`SELECT id, user_id, key_hash, label, scopes, created_at, expires_at, revoked
		 FROM api_keys WHERE key_hash = $1`, hash,
	).Scan(&k.ID, &k.UserID, &k.KeyHash, &k.Label, &k.Scopes, &k.CreatedAt, &k.ExpiresAt, &k.Revoked)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, errors.New("invalid API key")
		}
		return nil, nil, fmt.Errorf("validate api key: %w", err)
	}

	if k.Revoked {
		return nil, nil, errors.New("API key revoked")
	}
	if k.ExpiresAt != nil && k.ExpiresAt.Before(time.Now()) {
		return nil, nil, errors.New("API key expired")
	}

	user, err := s.GetUserByID(ctx, k.UserID)
	if err != nil {
		return nil, nil, err
	}
	if user == nil || !user.IsActive {
		return nil, nil, errors.New("user not found or inactive")
	}

	return user, k, nil
}

// RevokeAPIKey marks an API key as revoked.
func (s *AuthStore) RevokeAPIKey(ctx context.Context, keyID, userID int64) error {
	result, err := s.Pool.Exec(ctx,
		`UPDATE api_keys SET revoked = TRUE WHERE id = $1 AND user_id = $2`, keyID, userID)
	if err != nil {
		return fmt.Errorf("revoke api key: %w", err)
	}
	if result.RowsAffected() == 0 {
		return errors.New("API key not found")
	}
	return nil
}

// RollAPIKey revokes the existing key and creates a new one with the same label and scopes.
func (s *AuthStore) RollAPIKey(ctx context.Context, keyID, userID int64) (string, *APIKey, error) {
	// Fetch existing key to preserve label and scopes
	var label, scopes string
	err := s.Pool.QueryRow(ctx,
		`SELECT label, scopes FROM api_keys WHERE id = $1 AND user_id = $2 AND revoked = FALSE`,
		keyID, userID,
	).Scan(&label, &scopes)
	if err != nil {
		return "", nil, errors.New("API key not found or already revoked")
	}

	// Revoke the old key
	if _, err := s.Pool.Exec(ctx,
		`UPDATE api_keys SET revoked = TRUE WHERE id = $1`, keyID); err != nil {
		return "", nil, fmt.Errorf("roll api key (revoke): %w", err)
	}

	// Create replacement with same label and scopes
	return s.CreateAPIKey(ctx, userID, label, scopes)
}

// ListAPIKeys returns all API keys for a user.
func (s *AuthStore) ListAPIKeys(ctx context.Context, userID int64) ([]APIKey, error) {
	rows, err := s.Pool.Query(ctx,
		`SELECT id, user_id, key_hash, label, scopes, created_at, expires_at, revoked
		 FROM api_keys WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}
	defer rows.Close()

	var keys []APIKey
	for rows.Next() {
		var k APIKey
		if err := rows.Scan(&k.ID, &k.UserID, &k.KeyHash, &k.Label, &k.Scopes, &k.CreatedAt, &k.ExpiresAt, &k.Revoked); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, nil
}

// --- OIDC Provider operations ---

// ListOIDCProviders returns all OIDC provider configs from the database.
func (s *AuthStore) ListOIDCProviders(ctx context.Context) ([]OIDCProviderConfig, error) {
	rows, err := s.Pool.Query(ctx,
		`SELECT id, name, issuer_url, client_id, client_secret, redirect_uri, scopes, is_enabled, auto_register, default_role
		 FROM oidc_providers ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list oidc providers: %w", err)
	}
	defer rows.Close()

	var providers []OIDCProviderConfig
	for rows.Next() {
		var p OIDCProviderConfig
		if err := rows.Scan(&p.ID, &p.Name, &p.IssuerURL, &p.ClientID, &p.ClientSecret,
			&p.RedirectURI, &p.Scopes, &p.IsEnabled, &p.AutoRegister, &p.DefaultRole); err != nil {
			return nil, err
		}
		providers = append(providers, p)
	}
	return providers, nil
}

// CreateOIDCProvider inserts a new OIDC provider config.
func (s *AuthStore) CreateOIDCProvider(ctx context.Context, cfg OIDCProviderConfig) (*OIDCProviderConfig, error) {
	err := s.Pool.QueryRow(ctx,
		`INSERT INTO oidc_providers (name, issuer_url, client_id, client_secret, redirect_uri, scopes, is_enabled, auto_register, default_role)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING id`,
		cfg.Name, cfg.IssuerURL, cfg.ClientID, cfg.ClientSecret, cfg.RedirectURI,
		cfg.Scopes, cfg.IsEnabled, cfg.AutoRegister, cfg.DefaultRole,
	).Scan(&cfg.ID)
	if err != nil {
		return nil, fmt.Errorf("create oidc provider: %w", err)
	}
	return &cfg, nil
}

// DeleteOIDCProvider removes an OIDC provider config.
func (s *AuthStore) DeleteOIDCProvider(ctx context.Context, name string) error {
	result, err := s.Pool.Exec(ctx,
		`DELETE FROM oidc_providers WHERE name = $1`, name)
	if err != nil {
		return fmt.Errorf("delete oidc provider: %w", err)
	}
	if result.RowsAffected() == 0 {
		return errors.New("OIDC provider not found")
	}
	return nil
}
