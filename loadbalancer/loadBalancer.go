package loadbalancer

import (
	"goxy/config"
	"goxy/server"
)

// Represents a Load Balancer
type LoadBalancer interface {
	GetNext(ip string) (string, error)
	Finished(ip string)
}

func NewLoadBalancer(pool *server.ServerPool, config config.Loadbalancer) LoadBalancer {
	switch config.Algorithm {
	case "round_robin":
		return NewRoundRobin(pool)
	case "random":
		return NewRandom(pool)
	case "ipHash":
		return NewIpHash(pool)
	case "weighted":
		return NewWeighted(pool, config.Weights)
	case "weightedLeast":
		return NewWeightedLeast(pool, config.Weights)
	case "leastConnection":
		return NewLeastConnection(pool)
	default: // In case of invalid value, fallback to random
		return NewRandom(pool)
	}
}
