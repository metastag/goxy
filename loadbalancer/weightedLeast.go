package loadbalancer

import (
	"errors"
	"goxy/server"
	"sync"
)

// Represents a Weighted Least Connection System
type WeightedLeast struct {
	servers     *server.ServerPool
	connections map[string]int
	weights     map[string]int
	mu          sync.Mutex
}

// Returns a new WeightedLeast struct
func NewWeightedLeast(pool *server.ServerPool) *WeightedLeast {
	connections := make(map[string]int)
	weights := make(map[string]int) // fetch weights from config file (TODO)
	// when making the parser, ensure weights > 0
	weights["a"] = 5
	weights["b"] = 2
	weights["c"] = 1
	wl := WeightedLeast{servers: pool, connections: connections, weights: weights, mu: sync.Mutex{}}
	return &wl
}

// Implements Weighted Least Connection Algorithm
func (wl *WeightedLeast) GetNext(ip string) (string, error) {
	wl.mu.Lock()
	defer wl.mu.Unlock()

	servers := wl.servers.GetHealthyServers()

	if len(servers) == 0 {
		return "", errors.New("Load Balancer - No available server")
	}

	// Loop through the servers and find the lowest ratio
	minServer := servers[0]
	minValue := wl.connections[minServer] * wl.weights[minServer]

	for _, server := range servers[1:] {
		value := wl.connections[server] * wl.weights[server]
		if value < minValue {
			minServer = server
			minValue = value
		}
	}
	wl.connections[minServer]++ // Mark a new connection added
	return minServer, nil

}

// Mark request as completed
func (wl *WeightedLeast) Finished(ip string) {
	wl.mu.Lock()
	defer wl.mu.Unlock()

	if wl.connections[ip] > 0 {
		wl.connections[ip]--
	}
}
