// Command server runs RepPilot: JSON API + embedded web UI in one binary.
package main

import (
	"log"
	"net/http"
	"os"

	"reppilot/internal/drafter"
	"reppilot/internal/gbp"
	"reppilot/internal/server"
	"reppilot/internal/store"
	"reppilot/internal/wa"
	"reppilot/web"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8102"
	}

	st := store.Open("./data/store.json")
	srv := server.New(st, gbp.Mock{}, drafter.Template{}, wa.Mock{}, web.Files)

	addr := ":" + port
	log.Printf("RepPilot listening on http://localhost%s (gbp=mock, drafter=template, whatsapp=mock)", addr)
	if err := http.ListenAndServe(addr, srv.Handler()); err != nil {
		log.Fatal(err)
	}
}
