package webapp

import (
	"net/http"

	"github.com/gorilla/mux"
)

// Routes ...
func (app *App) Routes() *mux.Router {
	r := mux.NewRouter().StrictSlash(true)

	r.HandleFunc("/", app.Index).Methods("GET")
	r.HandleFunc("/", app.GetCode).Methods("POST")
	r.HandleFunc("/{code:[a-zA-Z0-9]{3,6}}", app.GetURL).Methods("GET")
	r.HandleFunc("/{code:[a-zA-Z0-9]{3,6}\\+}", app.GetStats).Methods("GET")

	// Error Handler
	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("<h1>404 Test!</h1>"))
	})

	fs := http.FileServer(http.Dir(app.StaticDir))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))

	return r
}
