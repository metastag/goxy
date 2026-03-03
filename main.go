package main

// make http server port a configurable option in yaml file

import (
	"goxy/health"
	"goxy/loadbalancer"
	"goxy/proxy"
	"goxy/ratelimit"
	"goxy/server"
	"log"
	"net/http"
)

func main() {
	serverPool := server.NewServerPool()
	healthChecker := health.NewHealthChecker(serverPool)
	loadBalancer := loadbalancer.NewLoadBalancer(serverPool)
	rateLimiter := ratelimit.NewRateLimiter()
	requestForwarder := proxy.NewRequestForwarder(loadBalancer, rateLimiter)

	// Ping servers periodically to test connection
	go healthChecker.Ping()

	// Fetch from config file (TODO)
	CERT_FILE := "./cert/cert.pem"
	KEY_FILE := "./cert/key.pem"

	// Start the server
	log.Println("Server starting on port 443")

	handler := http.HandlerFunc(requestForwarder.RequestHandler)
	err := http.ListenAndServeTLS(":443", CERT_FILE, KEY_FILE, handler)
	if err != nil {
		log.Println("Error starting the server - ", err)
	}
}
