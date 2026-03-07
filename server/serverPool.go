package server

import (
	"goxy/config"
	"log"
	"net/http"
	"net/url"
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
	pool    []string // ordered list of server URLs (matches config order)
	config  config.Servers
	mu      sync.RWMutex
}

// Initialize a new Server Pool
func NewServerPool(config config.Servers) *ServerPool {

	// Load ips and create a ServerStatus for them
	servers := make(map[string]*ServerStatus)
	for _, ip := range config.Pool {
		servers[ip] = &ServerStatus{healthy: true, errors: 0}
	}

	pool := ServerPool{servers: servers, pool: config.Pool, config: config, mu: sync.RWMutex{}}
	return &pool
}

// Returns a list of all healthy servers
func (sp *ServerPool) GetHealthyServers() []string {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	var pool []string
	for _, url := range sp.pool {
		if sp.servers[url].healthy {
			pool = append(pool, url)
		}
	}

	return pool
}

// Returns a list of all servers
func (sp *ServerPool) GetAllServers() []string {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	result := make([]string, len(sp.pool))
	copy(result, sp.pool)
	return result
}

func (sp *ServerPool) MarkError(url string) {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	s, ok := sp.servers[url]
	if !ok {
		return
	}
	s.errors++

	// If errors cross threshold, mark unhealthy
	if s.errors > sp.config.ErrorLimit {
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
		time.Sleep(time.Duration(sp.config.Ping) * time.Second)
		for _, server := range servers {
			pingURL, _ := url.JoinPath(server, "/ping")
			resp, err := httpClient.Get(pingURL)

			sp.mu.Lock()
			if err != nil || resp == nil || resp.StatusCode != 200 { // Ping failed
				log.Println("Ping failed for server - ", server)
				sp.servers[server].healthy = false // mark unhealthy
			} else {
				sp.servers[server].errors = 0
				sp.servers[server].healthy = true // mark healthy
			}
			if resp != nil {
				resp.Body.Close()
			}
			sp.mu.Unlock()
		}
	}
}
