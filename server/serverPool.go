package server

import (
	"log"
	"net/http"
	"sync"
	"time"
)

// Represents a server
type ServerStatus struct {
	healthy bool // whether server is available or not
	errors  int  // keep track of recent server errors
}

// Represents all servers
type ServerPool struct {
	servers map[string]*ServerStatus
	mu      sync.RWMutex
}

// Initialize a new Server Pool
func NewServerPool() *ServerPool {
	// Fetch server ip from config file (TODO)
	servers := make(map[string]*ServerStatus)
	servers["1"] = &ServerStatus{healthy: true, errors: 0}
	servers["2"] = &ServerStatus{healthy: true, errors: 0}
	servers["3"] = &ServerStatus{healthy: true, errors: 0}

	pool := ServerPool{servers: servers, mu: sync.RWMutex{}}
	return &pool
}

// Returns a list of all healthy servers
func (sp *ServerPool) GetHealthyServers() []string {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	var pool []string
	for url, server := range sp.servers {
		if server.healthy {
			pool = append(pool, url)
		}
	}

	return pool
}

// Returns a list of all servers
func (sp *ServerPool) GetAllServers() []string {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	var pool []string
	for url := range sp.servers {
		pool = append(pool, url)
	}
	return pool
}

func (sp *ServerPool) MarkError(url string) {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	s, ok := sp.servers[url]
	if !ok {
		return
	}
	s.errors++

	// Fetch the error limit from config file (TODO)
	// If errors cross threshold, mark unhealthy
	if s.errors > 100 {
		s.healthy = false
	}
}

// Periodically ping all servers
// If ping fails for a server, mark it as unhealthy
func (sp *ServerPool) Ping() {
	// Create an HTTP client with 20 second timeout to ping servers
	httpClient := http.Client{Timeout: 20 * time.Second}
	servers := sp.GetAllServers()

	for {
		time.Sleep(20 * time.Second) // Fetch time from config file (TODO)
		for _, url := range servers {
			resp, err := httpClient.Get(url + "/ping")

			sp.mu.Lock()
			if err != nil || resp == nil || resp.StatusCode != 200 { // Ping failed
				log.Println("Ping failed for server - ", url)
				sp.servers[url].healthy = false // mark unhealthy
			} else {
				sp.servers[url].errors = 0
				sp.servers[url].healthy = true // mark healthy
			}
			if resp != nil {
				resp.Body.Close()
			}
			sp.mu.Unlock()
		}
	}
}
