package main

// Represents a Load Balancer
type LoadBalancer struct {
	servers *ServerPool
	counter int
}

// Initialize a new Load Balancer
func NewLoadBalancer(pool *ServerPool) *LoadBalancer {
	loadBalancer := LoadBalancer{servers: pool, counter: 0}
	return &loadBalancer
}

// Implements Round Robin for now
func (lb *LoadBalancer) GetNext() string {
	servers := lb.servers.GetHealthyServers()

	if len(servers) == 0 { // Server list is empty
		return "None"
	}

	lb.counter += 1
	lb.counter = lb.counter % len(servers)
	return servers[lb.counter]
}
