package main

// make http server port a configurable option in yaml file

import (
	"goxy/health"
	"goxy/loadbalancer"
	"goxy/proxy"
	"goxy/server"
	"log"
	"net/http"
)

func main() {
	serverPool := server.NewServerPool()
	healthChecker := health.NewHealthChecker(serverPool)
	loadBalancer := loadbalancer.NewLoadBalancer(serverPool)
	requestForwarder := proxy.NewRequestForwarder(loadBalancer)

	// Ping servers periodically to test connection
	go healthChecker.Ping()

	// Fetch PORT from config file (TODO)
	PORT := ":" + "8080"

	// Start the server on PORT
	log.Println("Server starting on port", PORT)
	err := http.ListenAndServe(PORT, http.HandlerFunc(requestForwarder.RequestHandler))
	if err != nil {
		log.Println("Error starting the server - ", err)
	}
}
