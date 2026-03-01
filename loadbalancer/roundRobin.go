package loadbalancer

import (
	"errors"
	"goxy/server"
	"sync"
)

// Represents a Round Robin system
type RoundRobin struct {
	servers *server.ServerPool
	counter int
	mu      sync.Mutex
}

// Returns a new RoundRobin struct
func NewRoundRobin(pool *server.ServerPool) *RoundRobin {
	loadBalancer := RoundRobin{servers: pool, counter: 0, mu: sync.Mutex{}}
	return &loadBalancer
}

// Implements Round Robin
func (rr *RoundRobin) GetNext(ip string) (string, error) {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	servers := rr.servers.GetHealthyServers()

	if len(servers) == 0 { // If no available servers
		return "", errors.New("Load Balancer - No available server")
	}

	id := rr.counter % len(servers)
	rr.counter++
	s := servers[id]
	return s, nil
}

// Null function, to match the interface
func (rr *RoundRobin) Finished(ip string) {}
