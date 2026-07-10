package gateway

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

// workspaceJSON is the wire shape for a workspace.
type workspaceJSON struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	Plan      string `json:"plan"`
	IsOwner   bool   `json:"isOwner"`
	CreatedAt string `json:"createdAt"`
}

// handleWorkspaces handles GET (list accessible workspaces + active id) and
// POST (create a new workspace owned by the caller).
func (h *AdminHandler) handleWorkspaces(w http.ResponseWriter, r *http.Request) {
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
		workspaces, err := h.AuthStore.ListWorkspacesForUser(r.Context(), claims.UserID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list workspaces"})
			return
		}
		active, _ := h.AuthStore.ResolveActiveWorkspace(r.Context(), claims.UserID, claims.RegisteredClaims.ID, 0)
		rows := make([]workspaceJSON, 0, len(workspaces))
		for _, ws := range workspaces {
			rows = append(rows, workspaceJSON{
				ID:        ws.ID,
				Name:      ws.Name,
				Slug:      ws.Slug,
				Plan:      ws.Plan,
				IsOwner:   ws.OwnerID == claims.UserID,
				CreatedAt: ws.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
			})
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"workspaces":        rows,
			"activeWorkspaceId": active,
		})

	case http.MethodPost:
		var body struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}
		ws, err := h.AuthStore.CreateWorkspace(r.Context(), claims.UserID, body.Name)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, workspaceJSON{
			ID:        ws.ID,
			Name:      ws.Name,
			Slug:      ws.Slug,
			Plan:      ws.Plan,
			IsOwner:   true,
			CreatedAt: ws.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

// handleWorkspaceAction handles POST /api/v1/workspaces/switch and
// DELETE /api/v1/workspaces/{id}.
func (h *AdminHandler) handleWorkspaceAction(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	claims := h.requireAuth(w, r)
	if claims == nil {
		return
	}

	suffix := strings.TrimPrefix(r.URL.Path, "/api/v1/workspaces/")

	if suffix == "switch" {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		var body struct {
			WorkspaceID int64 `json:"workspaceId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}
		ok, err := h.AuthStore.UserCanAccessWorkspace(r.Context(), claims.UserID, body.WorkspaceID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to verify workspace access"})
			return
		}
		if !ok {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "you do not have access to that workspace"})
			return
		}
		if err := h.AuthStore.SetActiveWorkspace(r.Context(), claims.RegisteredClaims.ID, body.WorkspaceID); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to switch workspace"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"activeWorkspaceId": body.WorkspaceID})
		return
	}

	if r.Method != http.MethodDelete {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	id, err := strconv.ParseInt(suffix, 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid workspace id"})
		return
	}
	if err := h.AuthStore.DeleteWorkspace(r.Context(), claims.UserID, id); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
