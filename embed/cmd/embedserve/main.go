// Command embedserve serves the static directory embedded in the binary.
//
// Run it with:
//
//	go run ./cmd/embedserve
//
// then open http://localhost:8080/.
package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gz-zenn/go-annotations/embed"
)

func main() {
	handler, err := embed.StaticHandler()
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", handler)

	server := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Fatal(server.ListenAndServe())
}
