package server

import "sync"

// Represents a server
type Server struct {
	url     string
	healthy bool
}

// Represents all servers
type ServerPool struct {
	servers []Server
	mu      sync.RWMutex
}

// Initialize a new Server Pool
func NewServerPool() *ServerPool {
	// Fetch server ip from config file (TODO)
	s1 := Server{url: "1", healthy: true}
	s2 := Server{url: "2", healthy: true}
	s3 := Server{url: "3", healthy: true}
	servers := []Server{s1, s2, s3}

	pool := ServerPool{servers: servers, mu: sync.RWMutex{}}
	return &pool
}

// Sets health status of a server
func (sp *ServerPool) SetHealthy(url string, health bool) {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	for i := range sp.servers {
		if sp.servers[i].url == url {
			sp.servers[i].healthy = health
			return
		}
	}
}

// Returns a list of all healthy servers
func (sp *ServerPool) GetHealthyServers() []string {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	var pool []string
	for _, server := range sp.servers {
		if server.healthy {
			pool = append(pool, server.url)
		}
	}

	return pool
}

// Returns a list of all servers
func (sp *ServerPool) GetAllServers() []string {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	var pool []string
	for _, server := range sp.servers {
		pool = append(pool, server.url)
	}
	return pool
}
