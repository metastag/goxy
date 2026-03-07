package loadbalancer

import (
	"errors"
	"goxy/server"
	"sync"
)

// Represents a Weighted Round Robin system
type Weighted struct {
	servers *server.ServerPool
	weights map[string]int // maps server url to their weight
	counter int
	mu      sync.Mutex
}

// Returns a new Weighted struct
func NewWeighted(pool *server.ServerPool, weights []int) *Weighted {
	servers := pool.GetAllServers()

	// No. of weights should be same as no. of servers
	if len(weights) != len(servers) {
		return nil
	}

	// Map weights to server ips
	weightMap := make(map[string]int)
	for i, url := range servers {
		weightMap[url] = weights[i]
	}

	weighted := Weighted{servers: pool, weights: weightMap, counter: 0, mu: sync.Mutex{}}
	return &weighted
}

// Implements Weighted Round Robin
func (w *Weighted) GetNext(ip string) (string, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Get the list of healthy servers
	servers := w.servers.GetHealthyServers()

	if len(servers) == 0 { // If no available servers
		return "", errors.New("Load Balancer - No available server")
	}

	// Calculate total weight
	total := 0
	for _, server := range servers {
		total += w.weights[server]
	}

	// Set counter value
	id := w.counter % total
	w.counter++

	// Loop through servers, if the total weight gets bigger than id, that is the assigned server
	cumulative := 0
	for _, server := range servers {
		cumulative += w.weights[server]
		if id < cumulative {
			return server, nil
		}
	}
	return "", errors.New("Load Balancer - No available server")
}

// Null function, to match the interface
func (w *Weighted) Finished(ip string) {}
