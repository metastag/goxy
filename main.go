package main

// make http server port a configurable option in yaml file

import (
	"log"
	"net/http"
)

func main() {
	serverPool := NewServerPool()
	healthChecker := NewHealthChecker(serverPool)
	loadBalancer := NewLoadBalancer(serverPool)
	requestForwarder := NewRequestForwarder(loadBalancer)

	// Ping servers periodically to test connection
	go healthChecker.Ping()

	// Fetch PORT from config file (TODO)
	PORT := ":" + "8080"

	// Start the server on PORT
	log.Println("Server starting on port", PORT)
	err := http.ListenAndServe(PORT, http.HandlerFunc(requestForwarder.requestHandler))
	if err != nil {
		log.Println("Error starting the server - ", err)
	}
}
