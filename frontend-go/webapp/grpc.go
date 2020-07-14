package webapp

import (
	"context"
	"net/http"
	"regexp"

	"github.com/gorilla/mux"

	pb "github.com/incidrthreat/goshorten/frontend-go/pb"
)

// GetCode ...
func (app *App) GetCode(w http.ResponseWriter, r *http.Request) {
	c := pb.NewShortenerClient(app.Conn)

	resp, err := c.CreateURL(context.Background(), &pb.ShortURLReq{
		LongUrl: r.PostFormValue("url"),
	})

	if err != nil {
		app.RenderHTML(w, r, "index.html", &HTMLData{Data: "", Error: "Unable to store URL"})
		return
	}

	app.RenderHTML(w, r, "index.html", &HTMLData{Data: resp.ShortUrl, Error: ""})
}

// GetURL ...
func (app *App) GetURL(w http.ResponseWriter, r *http.Request) {
	validCode := regexp.MustCompile(`^[a-zA-Z0-9]{3,6}$`)
	vars := mux.Vars(r)
	code := vars["code"]

	if !validCode.MatchString(code) || code == "code" {
		app.RenderHTML(w, r, "index.html", &HTMLData{Data: "", Error: "Not valid URL Code"})
		return
	}

	c := pb.NewShortenerClient(app.Conn)
	resp, err := c.GetURL(context.Background(), &pb.URLReq{
		UrlCode: code,
	})

	if err != nil {
		app.RenderHTML(w, r, "index.html", &HTMLData{Data: "", Error: "URL expired"})
		return
	}

	http.Redirect(w, r, resp.RedirectUrl, 302)
}
