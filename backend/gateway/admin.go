package gateway

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/incidrthreat/goshorten/backend/auth"
	"github.com/incidrthreat/goshorten/backend/data"
)

// AdminHandler provides admin-only and self-service REST endpoints that live
// outside the gRPC-gateway (no proto changes needed).
type AdminHandler struct {
	AuthStore            *auth.AuthStore
	JWTMgr               *auth.JWTManager
	URLStore             data.URLStore
	OIDCMgr              *auth.OIDCManager
	// DisablePasswordLogin is true when the GOSHORTEN_DISABLE_PASSWORD_LOGIN env var
	// (or config.json flag) forces password login off. Admin cannot override it.
	DisablePasswordLogin bool
}

// requireAuth verifies the Bearer token and returns the user claims.
// Writes 401 and returns nil if the token is missing or invalid.
func (h *AdminHandler) requireAuth(w http.ResponseWriter, r *http.Request) *auth.Claims {
	hdr := r.Header.Get("Authorization")
	if !strings.HasPrefix(hdr, "Bearer ") {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return nil
	}
	claims, err := h.JWTMgr.Verify(strings.TrimPrefix(hdr, "Bearer "))
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid token"})
		return nil
	}
	return claims
}

// requireAdmin verifies the token and that the caller is an admin.
func (h *AdminHandler) requireAdmin(w http.ResponseWriter, r *http.Request) *auth.Claims {
	claims := h.requireAuth(w, r)
	if claims == nil {
		return nil
	}
	if claims.Role != "admin" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin access required"})
		return nil
	}
	return claims
}

// Register mounts all admin/self-service routes onto mux.
func (h *AdminHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/admin/users", h.handleUsers)
	mux.HandleFunc("/api/v1/admin/users/", h.handleUserByID)
	mux.HandleFunc("/api/v1/admin/short-urls", h.handleAdminURLs)
	mux.HandleFunc("/api/v1/admin/short-urls/", h.handleAdminURL)
	mux.HandleFunc("/api/v1/admin/oidc-providers", h.handleOIDCProviders)
	mux.HandleFunc("/api/v1/admin/oidc-providers/", h.handleOIDCProviderByName)
	mux.HandleFunc("/api/v1/auth/change-password", h.handleChangePassword)
	mux.HandleFunc("/api/v1/auth/profile", h.handleUpdateProfile)
	mux.HandleFunc("/api/v1/auth/account", h.handleAccount)
	mux.HandleFunc("/api/v1/auth/preferences", h.handlePreferences)
	mux.HandleFunc("/api/v1/auth/sessions", h.handleSessions)
	mux.HandleFunc("/api/v1/auth/sessions/", h.handleSessionByID)
	mux.HandleFunc("/api/v1/auth/sign-in-history", h.handleSignInHistory)
	mux.HandleFunc("/api/v1/auth/config", h.handleAuthConfig)
	mux.HandleFunc("/api/v1/admin/settings", h.handleAdminSettings)
}

// GET /api/v1/admin/users?search=&page=&page_size=
// POST /api/v1/admin/users — create a new local user.
func (h *AdminHandler) handleUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if h.requireAdmin(w, r) == nil {
		return
	}

	switch r.Method {
	case http.MethodGet:
		search := r.URL.Query().Get("search")
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
		if page < 1 {
			page = 1
		}
		if pageSize < 1 {
			pageSize = 20
		}

		users, total, err := h.AuthStore.ListUsers(r.Context(), search, page, pageSize)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list users"})
			return
		}

		type userRow struct {
			ID        int64  `json:"id"`
			Email     string `json:"email"`
			Name      string `json:"name"`
			Role      string `json:"role"`
			IsActive  bool   `json:"isActive"`
			CreatedAt string `json:"createdAt"`
		}
		rows := make([]userRow, 0, len(users))
		for _, u := range users {
			rows = append(rows, userRow{
				ID:        u.ID,
				Email:     u.Email,
				Name:      u.Name,
				Role:      u.Role,
				IsActive:  u.IsActive,
				CreatedAt: u.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
			})
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"users":    rows,
			"total":    total,
			"page":     page,
			"pageSize": pageSize,
		})

	case http.MethodPost:
		var body struct {
			Email    string `json:"email"`
			Name     string `json:"name"`
			Password string `json:"password"`
			Role     string `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}
		if body.Email == "" || body.Password == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email and password are required"})
			return
		}
		if body.Role == "" {
			body.Role = "user"
		}
		hash, err := auth.HashPassword(body.Password)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to hash password"})
			return
		}
		user, err := h.AuthStore.CreateUser(r.Context(), body.Email, body.Name, body.Role, &hash, nil, nil)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create user"})
			return
		}
		writeJSON(w, http.StatusCreated, map[string]interface{}{
			"id":        user.ID,
			"email":     user.Email,
			"name":      user.Name,
			"role":      user.Role,
			"isActive":  user.IsActive,
			"createdAt": user.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

// PATCH /api/v1/admin/users/{id} — update role/active/email/name.
// DELETE /api/v1/admin/users/{id} — permanently delete a user.
func (h *AdminHandler) handleUserByID(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if h.requireAdmin(w, r) == nil {
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/users/")
	userID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || userID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid user id"})
		return
	}

	switch r.Method {
	case http.MethodPatch:
		var body struct {
			Role     *string `json:"role"`
			IsActive *bool   `json:"isActive"`
			Email    *string `json:"email"`
			Name     *string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}

		params := auth.UpdateUserParams{
			Role:     body.Role,
			IsActive: body.IsActive,
			Email:    body.Email,
			Name:     body.Name,
		}
		user, err := h.AuthStore.UpdateUser(r.Context(), userID, params)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update user"})
			return
		}
		if user == nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"id":        user.ID,
			"email":     user.Email,
			"name":      user.Name,
			"role":      user.Role,
			"isActive":  user.IsActive,
			"createdAt": user.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		})

	case http.MethodDelete:
		if err := h.AuthStore.DeleteUser(r.Context(), userID); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete user"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

// GET /api/v1/admin/short-urls — full URL list (all owners) with createdByEmail.
func (h *AdminHandler) handleAdminURLs(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if h.requireAdmin(w, r) == nil {
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if h.URLStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "url store not configured"})
		return
	}

	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	pageSize, _ := strconv.Atoi(q.Get("pageSize"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	params := data.URLListParams{
		Page:     int32(page),
		PageSize: int32(pageSize),
		Search:   q.Get("search"),
		Tag:      q.Get("tag"),
		Domain:   q.Get("domain"),
		OrderBy:  q.Get("orderBy"),
		OrderDir: q.Get("orderDir"),
		// UserID intentionally nil — admins see all URLs regardless of owner
	}

	result, err := h.URLStore.List(params)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list urls"})
		return
	}

	type urlRow struct {
		Code           string   `json:"code"`
		LongUrl        string   `json:"longUrl"`
		Title          string   `json:"title"`
		IsActive       bool     `json:"isActive"`
		TotalClicks    int64    `json:"totalClicks"`
		Tags           []string `json:"tags"`
		CreatedAt      string   `json:"createdAt"`
		CreatedByEmail string   `json:"createdByEmail"`
		Domain         string   `json:"domain"`
	}
	rows := make([]urlRow, 0, len(result.URLs))
	for _, u := range result.URLs {
		domain := ""
		if u.Domain != nil {
			domain = *u.Domain
		}
		tags := u.Tags
		if tags == nil {
			tags = []string{}
		}
		rows = append(rows, urlRow{
			Code:           u.Code,
			LongUrl:        u.LongURL,
			Title:          u.Title,
			IsActive:       u.IsActive,
			TotalClicks:    u.TotalClicks,
			Tags:           tags,
			CreatedAt:      u.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
			CreatedByEmail: u.CreatedByEmail,
			Domain:         domain,
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"urls":     rows,
		"total":    result.Total,
		"page":     result.Page,
		"pageSize": result.PageSize,
	})
}

// GET  /api/v1/admin/short-urls/{code} — fetch a single URL with owner info.
// PATCH /api/v1/admin/short-urls/{code} — reassign URL owner (admin only).
func (h *AdminHandler) handleAdminURL(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if h.requireAdmin(w, r) == nil {
		return
	}
	if h.URLStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "url store not configured"})
		return
	}

	code := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/short-urls/")
	if code == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "code required"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		rec, err := h.URLStore.Get(code)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "url not found"})
			return
		}
		domain := ""
		if rec.Domain != nil {
			domain = *rec.Domain
		}
		tags := rec.Tags
		if tags == nil {
			tags = []string{}
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"code":           rec.Code,
			"longUrl":        rec.LongURL,
			"title":          rec.Title,
			"isActive":       rec.IsActive,
			"totalClicks":    rec.TotalClicks,
			"tags":           tags,
			"createdAt":      rec.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
			"createdByEmail": rec.CreatedByEmail,
			"createdByUserId": func() interface{} {
				if rec.CreatedByUserID != nil {
					return *rec.CreatedByUserID
				}
				return nil
			}(),
			"domain": domain,
		})

	case http.MethodPatch:
		var body struct {
			AssignedUserID *int64 `json:"assignedUserId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}
		if body.AssignedUserID == nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "assignedUserId is required"})
			return
		}
		_, err := h.URLStore.Update(data.URLUpdateParams{
			Code:           code,
			AssignedUserID: body.AssignedUserID,
		})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to reassign url"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

// POST /api/v1/auth/change-password
func (h *AdminHandler) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	claims := h.requireAuth(w, r)
	if claims == nil {
		return
	}

	var body struct {
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if body.NewPassword == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "newPassword is required"})
		return
	}

	user, err := h.AuthStore.GetUserByID(r.Context(), claims.UserID)
	if err != nil || user == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to fetch user"})
		return
	}

	// Verify current password (skip for admin changing own password only if they know it).
	if user.PasswordHash != nil && body.CurrentPassword != "" {
		if !auth.CheckPassword(body.CurrentPassword, *user.PasswordHash) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "current password is incorrect"})
			return
		}
	}

	hash, err := auth.HashPassword(body.NewPassword)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to hash password"})
		return
	}
	if err := h.AuthStore.UpdateUserPassword(r.Context(), claims.UserID, hash); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update password"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// GET /api/v1/auth/account — full account details including isOIDC and theme.
func (h *AdminHandler) handleAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	claims := h.requireAuth(w, r)
	if claims == nil {
		return
	}
	user, err := h.AuthStore.GetUserByID(r.Context(), claims.UserID)
	if err != nil || user == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to fetch account"})
		return
	}
	isOIDC := user.OIDCIssuer != nil && *user.OIDCIssuer != ""
	oidcProvider := ""
	if user.OIDCIssuer != nil {
		oidcProvider = *user.OIDCIssuer
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":           user.ID,
		"email":        user.Email,
		"name":         user.Name,
		"role":         user.Role,
		"isOIDC":       isOIDC,
		"oidcProvider": oidcProvider,
		"theme":        user.Theme,
	})
}

// PATCH /api/v1/auth/preferences — save user preferences (theme).
func (h *AdminHandler) handlePreferences(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPatch {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	claims := h.requireAuth(w, r)
	if claims == nil {
		return
	}
	var body struct {
		Theme *string `json:"theme"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if body.Theme != nil {
		switch *body.Theme {
		case "light", "dark", "system":
		default:
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "theme must be light, dark, or system"})
			return
		}
		if err := h.AuthStore.SetUserTheme(r.Context(), claims.UserID, *body.Theme); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save preferences"})
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// GET  /api/v1/auth/sessions   — list active sessions for the current user.
// DELETE /api/v1/auth/sessions — revoke all other sessions (sign out other devices).
func (h *AdminHandler) handleSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	claims := h.requireAuth(w, r)
	if claims == nil {
		return
	}

	switch r.Method {
	case http.MethodGet:
		sessions, err := h.AuthStore.ListSessions(r.Context(), claims.UserID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list sessions"})
			return
		}
		// Determine current session JTI from the Bearer token
		currentJTI := ""
		hdr := r.Header.Get("Authorization")
		if strings.HasPrefix(hdr, "Bearer ") {
			if c, err := h.JWTMgr.Verify(strings.TrimPrefix(hdr, "Bearer ")); err == nil {
				currentJTI = c.RegisteredClaims.ID
			}
		}
		type sessionRow struct {
			ID        string `json:"id"`
			CreatedAt string `json:"createdAt"`
			ExpiresAt string `json:"expiresAt"`
			IPAddress string `json:"ipAddress"`
			Label     string `json:"label"`
			IsCurrent bool   `json:"isCurrent"`
		}
		rows := make([]sessionRow, 0, len(sessions))
		for _, s := range sessions {
			rows = append(rows, sessionRow{
				ID:        s.ID,
				CreatedAt: s.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
				ExpiresAt: s.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z"),
				IPAddress: s.IPAddress,
				Label:     s.Label,
				IsCurrent: s.ID == currentJTI,
			})
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"sessions": rows})

	case http.MethodDelete:
		// Revoke all other sessions (keep the current one)
		currentJTI := ""
		hdr := r.Header.Get("Authorization")
		if strings.HasPrefix(hdr, "Bearer ") {
			if c, err := h.JWTMgr.Verify(strings.TrimPrefix(hdr, "Bearer ")); err == nil {
				currentJTI = c.RegisteredClaims.ID
			}
		}
		n, err := h.AuthStore.RevokeOtherSessions(r.Context(), claims.UserID, currentJTI)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to revoke sessions"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"revoked": n})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

// DELETE /api/v1/auth/sessions/{id} — revoke a specific session.
func (h *AdminHandler) handleSessionByID(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodDelete {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	claims := h.requireAuth(w, r)
	if claims == nil {
		return
	}
	sessionID := strings.TrimPrefix(r.URL.Path, "/api/v1/auth/sessions/")
	if sessionID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "session id required"})
		return
	}
	if err := h.AuthStore.RevokeSession(r.Context(), sessionID, claims.UserID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to revoke session"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// GET /api/v1/auth/sign-in-history — recent sign-in events for the current user.
func (h *AdminHandler) handleSignInHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	claims := h.requireAuth(w, r)
	if claims == nil {
		return
	}
	events, err := h.AuthStore.GetSignInHistory(r.Context(), claims.UserID, 20)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to fetch sign-in history"})
		return
	}
	type eventRow struct {
		IPAddress  string `json:"ipAddress"`
		UserAgent  string `json:"userAgent"`
		Success    bool   `json:"success"`
		SignedInAt string `json:"signedInAt"`
	}
	rows := make([]eventRow, 0, len(events))
	for _, e := range events {
		rows = append(rows, eventRow{
			IPAddress:  e.IPAddress,
			UserAgent:  e.UserAgent,
			Success:    e.Success,
			SignedInAt: e.SignedInAt.UTC().Format("2006-01-02T15:04:05Z"),
		})
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"events": rows})
}

// GET  /api/v1/admin/oidc-providers   — list all OIDC providers (admin).
// POST /api/v1/admin/oidc-providers   — create an OIDC provider (admin).
func (h *AdminHandler) handleOIDCProviders(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if h.requireAdmin(w, r) == nil {
		return
	}

	switch r.Method {
	case http.MethodGet:
		providers, err := h.AuthStore.ListOIDCProviders(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list providers"})
			return
		}
		type providerRow struct {
			ID           int64  `json:"id"`
			Name         string `json:"name"`
			IssuerURL    string `json:"issuerUrl"`
			ClientID     string `json:"clientId"`
			RedirectURI  string `json:"redirectUri"`
			Scopes       string `json:"scopes"`
			IsEnabled    bool   `json:"isEnabled"`
			AutoRegister bool   `json:"autoRegister"`
			DefaultRole  string `json:"defaultRole"`
		}
		rows := make([]providerRow, 0, len(providers))
		for _, p := range providers {
			rows = append(rows, providerRow{
				ID:           p.ID,
				Name:         p.Name,
				IssuerURL:    p.IssuerURL,
				ClientID:     p.ClientID,
				RedirectURI:  p.RedirectURI,
				Scopes:       p.Scopes,
				IsEnabled:    p.IsEnabled,
				AutoRegister: p.AutoRegister,
				DefaultRole:  p.DefaultRole,
			})
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"providers": rows})

	case http.MethodPost:
		var body struct {
			Name         string `json:"name"`
			IssuerURL    string `json:"issuerUrl"`
			ClientID     string `json:"clientId"`
			ClientSecret string `json:"clientSecret"`
			RedirectURI  string `json:"redirectUri"`
			Scopes       string `json:"scopes"`
			IsEnabled    bool   `json:"isEnabled"`
			AutoRegister bool   `json:"autoRegister"`
			DefaultRole  string `json:"defaultRole"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}
		if body.Name == "" || body.IssuerURL == "" || body.ClientID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name, issuerUrl, and clientId are required"})
			return
		}
		if body.Scopes == "" {
			body.Scopes = "openid email profile"
		}
		if body.DefaultRole == "" {
			body.DefaultRole = "user"
		}
		cfg := auth.OIDCProviderConfig{
			Name:         body.Name,
			IssuerURL:    body.IssuerURL,
			ClientID:     body.ClientID,
			ClientSecret: body.ClientSecret,
			RedirectURI:  body.RedirectURI,
			Scopes:       body.Scopes,
			IsEnabled:    body.IsEnabled,
			AutoRegister: body.AutoRegister,
			DefaultRole:  body.DefaultRole,
		}
		saved, err := h.AuthStore.CreateOIDCProvider(r.Context(), cfg)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create provider"})
			return
		}
		// Register live so OIDCAuthURL/OIDCCallback work without restart
		if h.OIDCMgr != nil {
			if err := h.OIDCMgr.RegisterProvider(r.Context(), *saved); err != nil {
				log.Warn("OIDC", "live registration failed (will work after restart)", "provider", saved.Name, "error", err)
			}
		}
		writeJSON(w, http.StatusCreated, map[string]interface{}{
			"id":           saved.ID,
			"name":         saved.Name,
			"issuerUrl":    saved.IssuerURL,
			"clientId":     saved.ClientID,
			"redirectUri":  saved.RedirectURI,
			"scopes":       saved.Scopes,
			"isEnabled":    saved.IsEnabled,
			"autoRegister": saved.AutoRegister,
			"defaultRole":  saved.DefaultRole,
		})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

// PATCH  /api/v1/admin/oidc-providers/{name} — update an OIDC provider.
// DELETE /api/v1/admin/oidc-providers/{name} — delete an OIDC provider.
func (h *AdminHandler) handleOIDCProviderByName(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if h.requireAdmin(w, r) == nil {
		return
	}
	name := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/oidc-providers/")
	if name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "provider name required"})
		return
	}

	switch r.Method {
	case http.MethodPatch:
		var body struct {
			IssuerURL    *string `json:"issuerUrl"`
			ClientID     *string `json:"clientId"`
			ClientSecret *string `json:"clientSecret"`
			RedirectURI  *string `json:"redirectUri"`
			Scopes       *string `json:"scopes"`
			IsEnabled    *bool   `json:"isEnabled"`
			AutoRegister *bool   `json:"autoRegister"`
			DefaultRole  *string `json:"defaultRole"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}
		params := auth.UpdateOIDCProviderParams{
			IssuerURL:    body.IssuerURL,
			ClientID:     body.ClientID,
			ClientSecret: body.ClientSecret,
			RedirectURI:  body.RedirectURI,
			Scopes:       body.Scopes,
			IsEnabled:    body.IsEnabled,
			AutoRegister: body.AutoRegister,
			DefaultRole:  body.DefaultRole,
		}
		saved, err := h.AuthStore.UpdateOIDCProvider(r.Context(), name, params)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update provider"})
			return
		}
		// Re-register live to pick up config changes
		if h.OIDCMgr != nil {
			h.OIDCMgr.UnregisterProvider(name)
			if regErr := h.OIDCMgr.RegisterProvider(r.Context(), *saved); regErr != nil {
				log.Warn("OIDC", "live re-registration failed", "provider", name, "error", regErr)
			}
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"id":           saved.ID,
			"name":         saved.Name,
			"issuerUrl":    saved.IssuerURL,
			"clientId":     saved.ClientID,
			"redirectUri":  saved.RedirectURI,
			"scopes":       saved.Scopes,
			"isEnabled":    saved.IsEnabled,
			"autoRegister": saved.AutoRegister,
			"defaultRole":  saved.DefaultRole,
		})

	case http.MethodDelete:
		if err := h.AuthStore.DeleteOIDCProvider(r.Context(), name); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete provider"})
			return
		}
		if h.OIDCMgr != nil {
			h.OIDCMgr.UnregisterProvider(name)
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

// PATCH /api/v1/auth/profile — self-service profile update (email and/or name).
func (h *AdminHandler) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPatch {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	claims := h.requireAuth(w, r)
	if claims == nil {
		return
	}

	var body struct {
		Email *string `json:"email"`
		Name  *string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	params := auth.UpdateUserParams{
		Email: body.Email,
		Name:  body.Name,
	}
	user, err := h.AuthStore.UpdateUser(r.Context(), claims.UserID, params)
	if err != nil || user == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update profile"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":    user.ID,
		"email": user.Email,
		"name":  user.Name,
		"role":  user.Role,
	})
}

// GET /api/v1/auth/config — public endpoint returning authentication configuration.
// Returns the effective password login setting (env var takes priority over DB).
func (h *AdminHandler) handleAuthConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	passwordEnabled := true
	envOverride := false

	if h.DisablePasswordLogin {
		// Hard-disabled via environment / config.json — cannot be overridden by admin
		passwordEnabled = false
		envOverride = true
	} else {
		val, err := h.AuthStore.GetSetting(r.Context(), "password_login_enabled")
		if err == nil && val == "false" {
			passwordEnabled = false
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"passwordLoginEnabled": passwordEnabled,
		"envOverride":          envOverride,
	})
}

// GET  /api/v1/admin/settings — returns current system settings (admin only).
// PATCH /api/v1/admin/settings — updates system settings (admin only).
func (h *AdminHandler) handleAdminSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if h.requireAdmin(w, r) == nil {
		return
	}

	switch r.Method {
	case http.MethodGet:
		passwordEnabled := true
		if h.DisablePasswordLogin {
			passwordEnabled = false
		} else {
			val, err := h.AuthStore.GetSetting(r.Context(), "password_login_enabled")
			if err == nil && val == "false" {
				passwordEnabled = false
			}
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"passwordLoginEnabled": passwordEnabled,
			"envOverride":          h.DisablePasswordLogin,
		})

	case http.MethodPatch:
		if h.DisablePasswordLogin {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "password login is controlled by the GOSHORTEN_DISABLE_PASSWORD_LOGIN environment variable and cannot be changed here",
			})
			return
		}
		var body struct {
			PasswordLoginEnabled *bool `json:"passwordLoginEnabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}
		if body.PasswordLoginEnabled != nil {
			val := "true"
			if !*body.PasswordLoginEnabled {
				val = "false"
			}
			if err := h.AuthStore.SetSetting(r.Context(), "password_login_enabled", val); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save setting"})
				return
			}
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
