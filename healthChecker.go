package main

import (
	"log"
	"net/http"
	"time"
)

// Represents a Periodic Health Checker system
type HealthChecker struct {
	servers    *ServerPool
	httpClient http.Client
}

// Initialize a new Health Checker
func NewHealthChecker(pool *ServerPool) *HealthChecker {
	httpClient := http.Client{Timeout: 20 * time.Second} // 20 second timeout for ping
	healthChecker := HealthChecker{servers: pool, httpClient: httpClient}

	return &healthChecker
}

// Periodically ping all servers
// If ping fails for a server, mark it as unhealthy
func (hc *HealthChecker) Ping() {
	servers := hc.servers.GetAllServers()

	for {
		time.Sleep(20 * time.Second) // Fetch time from config file (TODO)
		for _, server := range servers {
			resp, err := hc.httpClient.Get(server + "/ping")

			if err != nil || resp == nil || resp.StatusCode != 200 { // Ping failed
				log.Println("Ping failed for server - ", server)
				hc.servers.SetHealthy(server, false)
			} else {
				hc.servers.SetHealthy(server, true)
			}
			if resp != nil {
				resp.Body.Close()
			}
		}
	}
}
