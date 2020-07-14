package webapp

import (
	"html/template"
	"net/http"
	"path/filepath"
)

// HTMLData passing data to and from html
type HTMLData struct {
	Data  string
	Error string
}

// RenderHTML handles data
func (app *App) RenderHTML(w http.ResponseWriter, r *http.Request, page string, data *HTMLData) {
	files := []string{
		filepath.Join(app.HTMLDir, "base.html"),
		filepath.Join(app.HTMLDir, page),
		filepath.Join(app.HTMLDir, "footer.html"),
		filepath.Join(app.HTMLDir, "scripts.html"),
		filepath.Join(app.HTMLDir, "stylesheets.html"),
	}
	ts, err := template.ParseFiles(files...)

	if err != nil {
		app.ServerError(w, err)
	}

	err = ts.ExecuteTemplate(w, "base", data)
	if err != nil {
		app.ServerError(w, err)
	}
}

// DefaultHandler handles any requests not matched by a previous path
func (app *App) DefaultHandler(w http.ResponseWriter, r *http.Request) {
	app.NotFound(w)
}
