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
func NewWeightedLeast(pool *server.ServerPool, weights []int) *WeightedLeast {
	servers := pool.GetAllServers()
	if len(weights) != len(servers) {
		return nil
	}

	connections := make(map[string]int)

	// Map weights to server ips
	weightMap := make(map[string]int)
	for i, url := range servers {
		weightMap[url] = weights[i]
	}

	wl := WeightedLeast{servers: pool, connections: connections, weights: weightMap, mu: sync.Mutex{}}
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
