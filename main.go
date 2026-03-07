package main

import (
	"goxy/cache"
	"goxy/config"
	"goxy/loadbalancer"
	"goxy/proxy"
	"goxy/ratelimit"
	"goxy/server"
	"log"
	"net/http"
)

func main() {
	// Load config file
	config := config.LoadConfig()

	// Initialize a new server pool
	serverPool := server.NewServerPool(config.Servers)

	// Launch load balancer
	loadBalancer := loadbalancer.NewLoadBalancer(serverPool, config.Loadbalancer)

	// Initialize Rate Limiter
	var rateLimiter *ratelimit.RateLimiter
	if config.Ratelimiting.Enabled {
		rateLimiter = ratelimit.NewRateLimiter(config.Ratelimiting)
	} else {
		rateLimiter = nil
	}

	// Initialize Cache
	var cacheSystem *cache.Cache
	if config.Caching.Enabled {
		cacheSystem = cache.NewCache()
	} else {
		cacheSystem = nil
	}

	// Set up proxy
	requestForwarder := proxy.NewRequestForwarder(serverPool, loadBalancer, rateLimiter, cacheSystem)

	// Ping servers periodically to test connection
	go serverPool.Ping()

	handler := http.HandlerFunc(requestForwarder.RequestHandler)
	log.Println("Server starting on port 443")

	// Start the server in SSL mode or normal http mode
	if config.Certificate.Enabled {
		CERT_FILE := config.Certificate.CertFile
		KEY_FILE := config.Certificate.KeyFile

		err := http.ListenAndServeTLS(":443", CERT_FILE, KEY_FILE, handler)
		if err != nil {
			log.Println("Error starting the server - ", err)
		}
	} else {
		err := http.ListenAndServe(":443", handler)
		if err != nil {
			log.Println("Error starting the server - ", err)
		}

	}
}
