package webapp

import (
	"context"
	"net/http"
	"regexp"
	"strconv"

	"github.com/gorilla/mux"

	pb "github.com/incidrthreat/goshorten/frontend-go/pb"
)

// GetCode ...
func (app *App) GetCode(w http.ResponseWriter, r *http.Request) {
	validURL := regexp.MustCompile(`[(http(s)?):\/\/(www\.)?a-zA-Z0-9@:%._\+~#=]{2,256}\.[a-z]{2,6}\b([-a-zA-Z0-9@:%_\+.~#?&//=]*)`)

	if !validURL.MatchString(r.PostFormValue("url")) {
		app.RenderHTML(w, r, "index.html", &HTMLData{Error: "Not valid URL"})
		return
	}

	if r.PostFormValue("ttl") == "" {
		app.RenderHTML(w, r, "index.html", &HTMLData{Error: "Choose a TTL"})
		return
	}

	c := pb.NewShortenerClient(app.Conn)

	ttl64, err := strconv.ParseInt(r.PostFormValue("ttl"), 10, 64)
	if err != nil {
		app.RenderHTML(w, r, "index.html", &HTMLData{Error: "Could not convert TTL"})
		return
	}

	resp, err := c.CreateURL(context.Background(), &pb.URL{
		LongUrl: r.PostFormValue("url"),
		TTL:     ttl64,
	})

	if err != nil {
		app.RenderHTML(w, r, "index.html", &HTMLData{Error: "Unable to store URL"})
		return
	}

	app.RenderHTML(w, r, "index.html", &HTMLData{Data: resp.Code})
}

// GetURL ...
func (app *App) GetURL(w http.ResponseWriter, r *http.Request) {
	validCode := regexp.MustCompile(`^[a-zA-Z0-9]{3,6}$`)
	vars := mux.Vars(r)
	code := vars["code"]

	if !validCode.MatchString(code) || code == "code" {
		app.RenderHTML(w, r, "index.html", &HTMLData{Error: "Not valid URL Code"})
		return
	}

	c := pb.NewShortenerClient(app.Conn)
	resp, err := c.GetURL(context.Background(), &pb.Code{
		Code: code,
	})

	if err != nil {
		app.RenderHTML(w, r, "index.html", &HTMLData{Error: "URL expired"})
		return
	}

	prefix := regexp.MustCompile(`^https?://`)

	if !prefix.MatchString(resp.LongUrl) {
		http.Redirect(w, r, "http://"+resp.LongUrl, 308)
	} else {
		http.Redirect(w, r, resp.LongUrl, 308)
	}
}
