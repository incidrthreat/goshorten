package webapp

import (
	"net/http"
)

// Index ...
func (app *App) Index(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		url := r.PostFormValue("url")
		data := &HTMLData{
			Data:  url,
			Error: "",
		}
		app.RenderHTML(w, r, "index.html", data)
	} else {
		app.RenderHTML(w, r, "index.html", nil)
	}
}
