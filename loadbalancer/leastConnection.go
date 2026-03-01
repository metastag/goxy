package loadbalancer

import (
	"errors"
	"goxy/server"
	"sync"
)

// Represents a Least Connection system
type LeastConnection struct {
	servers     *server.ServerPool
	connections map[string]int
	mu          sync.Mutex
}

// Returns a new LeastConnection struct
func NewLeastConnection(pool *server.ServerPool) *LeastConnection {
	connections := make(map[string]int)
	lc := LeastConnection{servers: pool, connections: connections, mu: sync.Mutex{}}
	return &lc
}

// Implements Least Connection algorithm
func (lc *LeastConnection) GetNext(ip string) (string, error) {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	servers := lc.servers.GetHealthyServers()

	if len(servers) == 0 {
		return "", errors.New("Load Balancer - No available server")
	}

	// Loop through servers and find the one with least amount of connections
	minServer := servers[0]
	min := lc.connections[minServer]

	for _, server := range servers[1:] {
		if i := lc.connections[server]; i < min {
			min = i
			minServer = server
		}
	}
	lc.connections[minServer]++ // Mark a new connection added

	return minServer, nil
}

// Mark request as completed
func (lc *LeastConnection) Finished(ip string) {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	if lc.connections[ip] > 0 {
		lc.connections[ip] -= 1
	}
}
