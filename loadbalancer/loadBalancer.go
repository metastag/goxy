package loadbalancer

import "goxy/server"

// Represents a Load Balancer
type LoadBalancer interface {
	GetNext(ip string) (string, error)
	Finished(ip string)
}

func NewLoadBalancer(pool *server.ServerPool) LoadBalancer {
	return NewRoundRobin(pool) // fetch the type of algorithm from config file (TODO)
}
