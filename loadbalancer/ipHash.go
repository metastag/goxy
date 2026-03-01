package loadbalancer

import (
	"errors"
	"goxy/server"
	"hash/fnv"
)

// Represents a Source IP Hash System
type IpHash struct {
	servers *server.ServerPool
}

// Returns a new IPHash struct
func NewIpHash(pool *server.ServerPool) *IpHash {
	servers := IpHash{servers: pool}
	return &servers
}

// Implements Source IP Hash Algorithm
func (ih *IpHash) GetNext(ip string) (string, error) {
	// Load servers
	servers := ih.servers.GetHealthyServers()

	if len(servers) == 0 { // If no available server
		return "", errors.New("Load Balancer - No available server")
	}

	// Create a new hasher (32-bit FNV-1a hash)
	// Converts ip (string) -> id (int)
	hasher := fnv.New32a()
	hasher.Write([]byte(ip))
	id := hasher.Sum32()

	// The same ip on subsequent requests will be converted to the same id, leading to being mapped to the same server
	// Note that if some servers go down, it will change the length of available servers, and ips will be assigned to different servers
	// Once all servers become available again, the mapping will become normal again
	id = id % uint32(len(servers))
	return servers[id], nil
}

// Null function, to match the interface
func (ih *IpHash) Finished(ip string) {}
