package webapp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

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

	stats := Statistics{
		CreatedAt: time.Now().Format("Mon, 02 Jan 2006 15:04:05 MST"),
		Clicks:    "0",
	}

	out, err := json.Marshal(stats)
	if err != nil {
		fmt.Printf("JSON error: %v", err)
	}

	resp, err := c.CreateURL(context.Background(), &pb.URL{
		LongUrl: r.PostFormValue("url"),
		TTL:     ttl64,
		Stats:   string(out),
	})

	if err != nil {
		app.RenderHTML(w, r, "index.html", &HTMLData{Error: "Unable to store URL"})
	}

	app.RenderHTML(w, r, "index.html", &HTMLData{Data: resp.Code})
}

// GetURL ...
func (app *App) GetURL(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	code := vars["code"]

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

// GetStats ...
func (app *App) GetStats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	code := vars["code"]

	c := pb.NewShortenerClient(app.Conn)
	resp, err := c.GetStats(context.Background(), &pb.Code{
		Code: strings.TrimSuffix(code, "+"),
	})

	var stats Statistics
	err = json.Unmarshal([]byte(resp.Stats), &stats)
	if err != nil {
		fmt.Printf("JSON Unmarshal Error: %v", err)
	}

	if stats.CreatedAt == "" {
		app.RenderHTML(w, r, "index.html", &HTMLData{Error: "URL Expired without Statistics"})
		return
	}

	app.RenderHTML(w, r, "stats.html", &HTMLData{
		Stats: Statistics{
			URL:          stats.URL,
			Code:         stats.Code,
			CreatedAt:    stats.CreatedAt,
			Clicks:       stats.Clicks,
			LastAccessed: stats.LastAccessed,
		},
	})
}
