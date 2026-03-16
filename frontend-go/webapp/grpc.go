package webapp

import (
	"context"
	"net/http"
	"regexp"
	"strings"

	pb "github.com/incidrthreat/goshorten/frontend-go/pb"
)

// GetURL handles short-code redirect requests.
func (app *App) GetURL(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimPrefix(r.URL.Path, "/")

	c := pb.NewShortenerClient(app.Conn)
	resp, err := c.GetURL(context.Background(), &pb.GetURLRequest{
		Code:           code,
		VisitorIp:      extractIP(r),
		VisitorUa:      r.UserAgent(),
		VisitorReferer: r.Referer(),
	})

	if err != nil {
		// Code not found → send to SPA homepage
		http.Redirect(w, r, "/?not_found="+code, http.StatusTemporaryRedirect)
		return
	}

	prefix := regexp.MustCompile(`^https?://`)
	redirectURL := resp.LongUrl
	if !prefix.MatchString(redirectURL) {
		redirectURL = "http://" + redirectURL
	}

	// Use the redirect type from the stored URL (default 302)
	redirectType := int(resp.RedirectType)
	if redirectType == 0 {
		redirectType = 302
	}

	// Add X-Robots-Tag header if not crawlable
	if !resp.IsCrawlable {
		w.Header().Set("X-Robots-Tag", "noindex")
	}

	http.Redirect(w, r, redirectURL, redirectType)
}

// extractIP returns the client IP from X-Forwarded-For or RemoteAddr.
func extractIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
	if xri := r.Header.Get("X-Real-Ip"); xri != "" {
		return strings.TrimSpace(xri)
	}
	host := r.RemoteAddr
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
	}
	return host
}
