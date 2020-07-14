package main

import (
	"log"
	"net/http"

	"github.com/incidrthreat/goshorten/frontend-go/webapp"
	"google.golang.org/grpc"
)

const (
	port      string = ":8081"
	htmlDir   string = "./ui/templates"
	staticDir string = "./ui/static"
)

func main() {
	conn, err := grpc.Dial("grpcbackend:9000", grpc.WithInsecure())
	// TODO: Better error handling and keep-alive
	if err != nil {
		log.Println(err)
	}
	defer conn.Close()

	app := &webapp.App{
		HTMLDir:   htmlDir,
		StaticDir: staticDir,
		Conn:      conn,
	}

	log.Printf("Starting URL Shortener on port %s", port)

	err = http.ListenAndServe(port, app.Routes())
	log.Fatal(err)

}
