package gateway

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/incidrthreat/goshorten/backend/auth"
)

// --- shared helpers -------------------------------------------------------

// pathWorkspaceID parses the {id} path value.
func pathWorkspaceID(r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

// requireWorkspaceMember verifies the caller can access the workspace and returns
// (claims, role). Writes an error and returns nil claims otherwise.
func (h *AdminHandler) requireWorkspaceMember(w http.ResponseWriter, r *http.Request, workspaceID int64) (*auth.Claims, string) {
	claims := h.requireAuth(w, r)
	if claims == nil {
		return nil, ""
	}
	// Platform operators (global admins) have an override across workspaces.
	if claims.Role == "admin" {
		role, err := h.AuthStore.WorkspaceRole(r.Context(), claims.UserID, workspaceID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve role"})
			return nil, ""
		}
		if role == "" {
			role = auth.RoleAdmin // operator acts with admin-level workspace rights
		}
		return claims, role
	}
	role, err := h.AuthStore.WorkspaceRole(r.Context(), claims.UserID, workspaceID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve role"})
		return nil, ""
	}
	if role == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "you are not a member of this workspace"})
		return nil, ""
	}
	return claims, role
}

// requireWorkspaceManager additionally requires owner/admin (management) rights.
func (h *AdminHandler) requireWorkspaceManager(w http.ResponseWriter, r *http.Request, workspaceID int64) (*auth.Claims, string) {
	claims, role := h.requireWorkspaceMember(w, r, workspaceID)
	if claims == nil {
		return nil, ""
	}
	if !auth.RoleAtLeast(role, auth.RoleAdmin) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "workspace admin or owner access required"})
		return nil, ""
	}
	return claims, role
}

func (h *AdminHandler) inviteAcceptURL(token string) string {
	base := strings.TrimRight(h.AppBaseURL, "/")
	return base + "/invite/" + token
}

// --- members --------------------------------------------------------------

type memberJSON struct {
	UserID   int64  `json:"userId"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Role     string `json:"role"`
	Status   string `json:"status"`
	JoinedAt string `json:"joinedAt"`
}

type invitationJSON struct {
	ID        int64  `json:"id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	Status    string `json:"status"`
	InvitedBy string `json:"invitedBy"`
	ExpiresAt string `json:"expiresAt"`
	CreatedAt string `json:"createdAt"`
	AcceptURL string `json:"acceptUrl,omitempty"`
}

// GET /api/v1/workspaces/{id}/members — list members, pending invites, and the
// caller's own role. Any member may view.
func (h *AdminHandler) handleMembers(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	workspaceID, ok := pathWorkspaceID(r)
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid workspace id"})
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	claims, role := h.requireWorkspaceMember(w, r, workspaceID)
	if claims == nil {
		return
	}

	members, err := h.AuthStore.ListMembers(r.Context(), workspaceID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list members"})
		return
	}
	memberRows := make([]memberJSON, 0, len(members))
	for _, m := range members {
		memberRows = append(memberRows, memberJSON{
			UserID:   m.UserID,
			Email:    m.Email,
			Name:     m.Name,
			Role:     m.Role,
			Status:   m.Status,
			JoinedAt: m.CreatedAt.UTC().Format(time.RFC3339),
		})
	}

	// Only managers see pending invitations (which reveal emails).
	inviteRows := []invitationJSON{}
	if auth.RoleAtLeast(role, auth.RoleAdmin) {
		invites, err := h.AuthStore.ListInvitations(r.Context(), workspaceID, true)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list invitations"})
			return
		}
		inviteRows = h.invitationRows(invites, false)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"members":        memberRows,
		"pendingInvites": inviteRows,
		"yourRole":       role,
	})
}

// PATCH  /api/v1/workspaces/{id}/members/{userId} — change a member's role.
// DELETE /api/v1/workspaces/{id}/members/{userId} — remove a member (self-leave allowed).
func (h *AdminHandler) handleMemberByID(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	workspaceID, ok := pathWorkspaceID(r)
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid workspace id"})
		return
	}
	targetUserID, err := strconv.ParseInt(r.PathValue("userId"), 10, 64)
	if err != nil || targetUserID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid user id"})
		return
	}

	switch r.Method {
	case http.MethodPatch:
		claims, _ := h.requireWorkspaceManager(w, r, workspaceID)
		if claims == nil {
			return
		}
		var body struct {
			Role string `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}
		if body.Role == auth.RoleOwner {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "use transfer-ownership to assign the owner role"})
			return
		}
		if !auth.ValidWorkspaceRole(body.Role) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid role"})
			return
		}
		if err := h.AuthStore.UpdateMemberRole(r.Context(), workspaceID, targetUserID, body.Role); err != nil {
			h.writeMemberErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})

	case http.MethodDelete:
		claims := h.requireAuth(w, r)
		if claims == nil {
			return
		}
		// Members may remove themselves (leave); otherwise management is required.
		if claims.UserID != targetUserID {
			if mgr, _ := h.requireWorkspaceManager(w, r, workspaceID); mgr == nil {
				return
			}
		}
		if err := h.AuthStore.RemoveMember(r.Context(), workspaceID, targetUserID); err != nil {
			h.writeMemberErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

// POST /api/v1/workspaces/{id}/transfer-ownership — hand the owner role to another
// member. Only the current owner (or a platform admin) may do this.
func (h *AdminHandler) handleTransferOwnership(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	workspaceID, ok := pathWorkspaceID(r)
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid workspace id"})
		return
	}
	claims, role := h.requireWorkspaceMember(w, r, workspaceID)
	if claims == nil {
		return
	}
	if role != auth.RoleOwner && claims.Role != "admin" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "only the workspace owner can transfer ownership"})
		return
	}
	var body struct {
		NewOwnerUserID int64 `json:"newOwnerUserId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.NewOwnerUserID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "newOwnerUserId is required"})
		return
	}

	ws, err := h.AuthStore.GetWorkspace(r.Context(), workspaceID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "workspace not found"})
		return
	}
	if err := h.AuthStore.TransferOwnership(r.Context(), workspaceID, ws.OwnerID, body.NewOwnerUserID); err != nil {
		h.writeMemberErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- invitations ----------------------------------------------------------

// GET  /api/v1/workspaces/{id}/invitations — list pending invites (manager).
// POST /api/v1/workspaces/{id}/invitations — create an invite (manager).
func (h *AdminHandler) handleInvitations(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	workspaceID, ok := pathWorkspaceID(r)
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid workspace id"})
		return
	}
	claims, _ := h.requireWorkspaceManager(w, r, workspaceID)
	if claims == nil {
		return
	}

	switch r.Method {
	case http.MethodGet:
		invites, err := h.AuthStore.ListInvitations(r.Context(), workspaceID, false)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list invitations"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"invitations": h.invitationRows(invites, false)})

	case http.MethodPost:
		var body struct {
			Email string `json:"email"`
			Role  string `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}
		inv, err := h.AuthStore.CreateInvitation(r.Context(), workspaceID, body.Email, body.Role, claims.UserID)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		h.sendInviteEmail(r, inv, claims.Email)
		row := h.invitationRow(inv, true)
		writeJSON(w, http.StatusCreated, row)

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

// DELETE /api/v1/workspaces/{id}/invitations/{invId} — revoke a pending invite.
func (h *AdminHandler) handleInvitationByID(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodDelete {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	workspaceID, ok := pathWorkspaceID(r)
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid workspace id"})
		return
	}
	invID, err := strconv.ParseInt(r.PathValue("invId"), 10, 64)
	if err != nil || invID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid invitation id"})
		return
	}
	if claims, _ := h.requireWorkspaceManager(w, r, workspaceID); claims == nil {
		return
	}
	if err := h.AuthStore.RevokeInvitation(r.Context(), workspaceID, invID); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// POST /api/v1/workspaces/{id}/invitations/{invId}/resend — refresh token & resend.
func (h *AdminHandler) handleInvitationResend(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	workspaceID, ok := pathWorkspaceID(r)
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid workspace id"})
		return
	}
	invID, err := strconv.ParseInt(r.PathValue("invId"), 10, 64)
	if err != nil || invID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid invitation id"})
		return
	}
	claims, _ := h.requireWorkspaceManager(w, r, workspaceID)
	if claims == nil {
		return
	}
	inv, err := h.AuthStore.ResendInvitation(r.Context(), workspaceID, invID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	h.sendInviteEmail(r, inv, claims.Email)
	writeJSON(w, http.StatusOK, h.invitationRow(inv, true))
}

// GET /api/v1/invitations/{token} — public preview of an invite so the accept page
// can render the workspace name and decide whether the user must sign up first.
func (h *AdminHandler) handleInvitePreview(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	token := r.PathValue("token")
	inv, err := h.AuthStore.GetInvitationByToken(r.Context(), token)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	existing, _ := h.AuthStore.GetUserByEmail(r.Context(), inv.Email)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"email":                inv.Email,
		"role":                 inv.Role,
		"workspaceName":        inv.WorkspaceName,
		"hasAccount":           existing != nil,
		"passwordLoginEnabled": h.passwordLoginEnabled(r),
		"expiresAt":            inv.ExpiresAt.UTC().Format(time.RFC3339),
	})
}

// POST /api/v1/invitations/{token}/accept — accept an invite (Phase 15.7/15.8).
// Cases:
//   - Authenticated user whose email matches → join immediately.
//   - No account for the email → create one from {name,password} and issue a token.
//   - Account exists but caller not authenticated → 401 needsLogin.
func (h *AdminHandler) handleInviteAccept(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	token := r.PathValue("token")
	inv, err := h.AuthStore.GetInvitationByToken(r.Context(), token)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	// Authenticated path: the Bearer identity must match the invited email.
	if hdr := r.Header.Get("Authorization"); strings.HasPrefix(hdr, "Bearer ") {
		claims, verr := h.JWTMgr.Verify(strings.TrimPrefix(hdr, "Bearer "))
		if verr != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid token"})
			return
		}
		if _, err := h.AuthStore.AcceptInvitation(r.Context(), token, claims.UserID, claims.Email); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":      "accepted",
			"workspaceId": inv.WorkspaceID,
		})
		return
	}

	// Unauthenticated path.
	existing, err := h.AuthStore.GetUserByEmail(r.Context(), inv.Email)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to look up account"})
		return
	}
	if existing != nil {
		// Account exists — tell the client to log in and retry with a token.
		writeJSON(w, http.StatusUnauthorized, map[string]interface{}{
			"error":      "please sign in to accept this invitation",
			"needsLogin": true,
			"email":      inv.Email,
		})
		return
	}

	// New account (Phase 15.8): self-service signup for the invited email.
	if !h.passwordLoginEnabled(r) {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "password signup is disabled on this instance; ask an administrator to create your account",
		})
		return
	}
	var body struct {
		Name     string `json:"name"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Password) < 8 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "a password of at least 8 characters is required"})
		return
	}
	hash, err := auth.HashPassword(body.Password)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to hash password"})
		return
	}
	user, err := h.AuthStore.CreateUser(r.Context(), inv.Email, body.Name, "user", &hash, nil, nil)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create account"})
		return
	}
	if _, err := h.AuthStore.AcceptInvitation(r.Context(), token, user.ID, user.Email); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	// Log the new user straight in.
	tokenStr, jti, err := h.JWTMgr.Generate(user.ID, user.Email, user.Role)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "account created but sign-in failed; please log in"})
		return
	}
	expiry := time.Now().Add(time.Duration(h.JWTMgr.ExpiryHr) * time.Hour)
	ip, ua := clientIPFromHTTP(r), r.UserAgent()
	_ = h.AuthStore.CreateSession(r.Context(), jti, user.ID, ip, ua, ua, expiry)
	h.AuthStore.LogSignIn(r.Context(), &user.ID, ip, ua, true)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":      "accepted",
		"workspaceId": inv.WorkspaceID,
		"token":       tokenStr,
		"user": map[string]interface{}{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
			"role":  user.Role,
		},
	})
}

// --- helpers --------------------------------------------------------------

func (h *AdminHandler) sendInviteEmail(r *http.Request, inv *auth.Invitation, inviterEmail string) {
	if h.Mailer == nil {
		return
	}
	wsName := inv.WorkspaceName
	if wsName == "" {
		if ws, err := h.AuthStore.GetWorkspace(r.Context(), inv.WorkspaceID); err == nil {
			wsName = ws.Name
		}
	}
	if err := h.Mailer.SendInvitation(inv.Email, wsName, inviterEmail, h.inviteAcceptURL(inv.Token)); err != nil {
		log.Warn("Mail", "failed to send invitation", "to", inv.Email, "error", err)
	}
}

func (h *AdminHandler) invitationRow(inv *auth.Invitation, includeURL bool) invitationJSON {
	row := invitationJSON{
		ID:        inv.ID,
		Email:     inv.Email,
		Role:      inv.Role,
		Status:    inv.Status,
		InvitedBy: inv.InvitedByEmail,
		ExpiresAt: inv.ExpiresAt.UTC().Format(time.RFC3339),
		CreatedAt: inv.CreatedAt.UTC().Format(time.RFC3339),
	}
	// The accept URL is only surfaced to managers (create/resend responses) so they
	// can copy a link when SMTP is not configured.
	if includeURL {
		row.AcceptURL = h.inviteAcceptURL(inv.Token)
	}
	return row
}

func (h *AdminHandler) invitationRows(invs []auth.Invitation, includeURL bool) []invitationJSON {
	rows := make([]invitationJSON, 0, len(invs))
	for i := range invs {
		rows = append(rows, h.invitationRow(&invs[i], includeURL))
	}
	return rows
}

func (h *AdminHandler) writeMemberErr(w http.ResponseWriter, err error) {
	if errors.Is(err, auth.ErrLastOwner) {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
}

// clientIPFromHTTP extracts the best-effort client IP from an HTTP request.
func clientIPFromHTTP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.IndexByte(xff, ','); i >= 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	host := r.RemoteAddr
	if i := strings.LastIndexByte(host, ':'); i >= 0 {
		host = host[:i]
	}
	return host
}
