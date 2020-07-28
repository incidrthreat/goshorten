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
	r.HandleFunc("/{code}", app.GetURL).Methods("GET")

	fs := http.FileServer(http.Dir(app.StaticDir))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))

	return r
}
