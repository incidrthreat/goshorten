package webapp

import (
	"net/http"
)

// Index ...
func (app *App) Index(w http.ResponseWriter, r *http.Request) {
	app.RenderHTML(w, r, "index.html", nil)
}
