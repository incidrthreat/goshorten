package webapp

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

// APIProxy returns a reverse proxy handler that forwards /api/* to the backend gateway.
func APIProxy(backendURL string) http.Handler {
	target, err := url.Parse(backendURL)
	if err != nil {
		panic("invalid backend URL: " + err.Error())
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	// Preserve the original director but set the host
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = target.Host
	}

	return proxy
}
