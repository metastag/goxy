![logo](gopher.jpg)


### goxy

a reverse proxy implementation written in golang

##### Features
1. Request Forwarding
2. Periodic Health Checks
3. Load Balancing (7 algorithms to choose from)
4. SSL/TLS Termination
5. Rate Limiting
6. Caching
7. Circuit Breaking
8. TOML configuration file

#### Usage
1. Clone this repo, you should have Go installed and optionally Docker for testing
```bash
git clone https://github.com/metastag/goxy.git
cd goxy
```

2. You can use Docker to simulate backend servers for testing, example in `/testing` folder
```bash
cd testing
docker build -t testing .
docker compose up
```

3. Configure settings in `config.toml`, there is a fallback config in `/config` folder

4. Start the proxy from project root, `go run main.go`

5. The proxy always runs on port 443

---

#### Implementation Details
**1. Request Forwarding**
- Handles timeouts, hop-by-hop headers
- Adds proxy-specific headers

**2. Periodic Health Checks**
- Periodically pings backend servers on `/ping`, if it does not recieve `200` response, marks them as unhealthy

**3. Load Balancing**
- Round Robin, Random, Source IP Hash, Weighted, Least Weighted, Least Connection implemented

**4. SSL/TLS Termination**
- Link certificate + key file in `config.toml`

**5. Rate Limiting**
- Implements Token Bucket algorithm for rate limiting
- Has a global bucket (to define total rate limit, like 1000 req/min)
- Has buckets per ip (so that one ip cannot hog the servers)
- TTL cleanup every 5 minutes

**6. Caching**
- Parses Cache-Control headers
- Also Implements Etag validation, Vary headers
- Removes resources whose `max-age` has been crossed by 1 hour

**7. Circuit Breaking**
- Keeps track of backend internal server errors
- If the errors cross a limit, mark the server as unhealthy

**8. TOML Configuration File**
- Configure backend server ips, set ping time
- Choose the load balancing algorithm (default is random)
- SSL/caching/Rate limiting can be toggled on/off