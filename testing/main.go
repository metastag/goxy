package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

var addr string

func requestHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "hello from server %s", addr)
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr = "localhost:" + port

	log.Printf("Server starting on %s", addr)

	handler := http.HandlerFunc(requestHandler)
	http.Handle("/ping", handler)

	err := http.ListenAndServe(":"+port, handler)
	if err != nil {
		log.Fatal("Server crashed - ", err)
	}
}
