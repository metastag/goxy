package loadbalancer

import (
	"errors"
	"goxy/server"
	"math/rand/v2"
)

// Represents Random Load Balancing System
type Random struct {
	servers *server.ServerPool
}

// Returns a new Random struct
func NewRandom(pool *server.ServerPool) *Random {
	random := Random{servers: pool}
	return &random
}

// Implements Random Load Balancing
func (r *Random) GetNext(ip string) (string, error) {
	servers := r.servers.GetHealthyServers()

	if len(servers) == 0 {
		return "", errors.New("Load Balancer - No available server")
	}

	// Choose a server at random
	id := rand.IntN(len(servers))

	return servers[id], nil
}

// Null function, to match the interface
func (r *Random) Finished(ip string) {}
