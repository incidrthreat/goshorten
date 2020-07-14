package webapp

import (
	"log"
	"net/http"
	"runtime/debug"
)

// ServerError helper writes an error message and stack trace to the log, then
// sends a generic 500 Internal Server Error response to the user.
func (app *App) ServerError(w http.ResponseWriter, err error) {
	log.Printf("%s\n%s", err.Error(), debug.Stack())
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}

// ClientError helper sends a specific status code and corresponding description
// to the user.
func (app *App) ClientError(w http.ResponseWriter, status int) {
	http.Error(w, http.StatusText(status), status)
}

// NotFound helper. This is simply a convenience wrapper around ClientError
// which sends a 404 Not Found response to the user.
func (app *App) NotFound(w http.ResponseWriter) {
	app.ClientError(w, http.StatusNotFound)
}

// Forbidden writes a 403 forbidden to the passed responsewriter
func (app *App) Forbidden(w http.ResponseWriter) {
	app.ClientError(w, http.StatusForbidden)
}
