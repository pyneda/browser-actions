package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8765", "listen address")
	dir := flag.String("dir", "examples/fixtures", "directory to serve")
	flag.Parse()

	mux := http.NewServeMux()
	fileServer := http.FileServer(http.Dir(*dir))
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		fileServer.ServeHTTP(w, r)
	}))

	srv := &http.Server{
		Addr:              *addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	fmt.Printf("browser-actions fixture server listening on http://%s\n", *addr)
	fmt.Printf("serving directory: %s\n", *dir)
	log.Fatal(srv.ListenAndServe())
}
