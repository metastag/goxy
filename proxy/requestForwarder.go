package proxy

import (
	"encoding/json"
	"goxy/cache"
	"goxy/loadbalancer"
	"goxy/ratelimit"
	"goxy/server"
	"io"
	"log"
	"net"
	"net/http"
	"time"
)

// Represents a Request Forwarding system
type RequestForwarder struct {
	sp              *server.ServerPool
	lb              loadbalancer.LoadBalancer
	rl              *ratelimit.RateLimiter
	c               *cache.Cache
	httpClient      http.Client
	hopByHopHeaders map[string]bool
}

// Initialize a new Request Forwarder
func NewRequestForwarder(sp *server.ServerPool, lb loadbalancer.LoadBalancer, rl *ratelimit.RateLimiter, c *cache.Cache) *RequestForwarder {
	httpClient := http.Client{Timeout: 20 * time.Second}
	hopByHopHeaders := map[string]bool{
		"Connection":          true,
		"Keep-Alive":          true,
		"Proxy-Authenticate":  true,
		"Proxy-Authorization": true,
		"TE":                  true,
		"Trailers":            true,
		"Transfer-Encoding":   true,
		"Upgrade":             true,
	}
	requestForwarder := RequestForwarder{
		sp:              sp,
		lb:              lb,
		rl:              rl,
		c:               c,
		httpClient:      httpClient,
		hopByHopHeaders: hopByHopHeaders,
	}
	return &requestForwarder
}

func (rf *RequestForwarder) constructRequest(r *http.Request) (*http.Request, string, int) {
	// Construct proxy url
	backendURL, err := rf.lb.GetNext(r.Host + r.URL.String()) // Load balancer assigns a backend server
	if err != nil {
		log.Println("Error while creating proxy request - ", err)
		return nil, "", 503
	}

	forwardUrl := backendURL + r.URL.Path

	// Add any query parameters to path
	if r.URL.RawQuery != "" {
		forwardUrl += "?" + r.URL.RawQuery
	}

	// Generate new request with same method and body
	proxyRequest, err := http.NewRequest(r.Method, forwardUrl, r.Body)
	if err != nil {
		log.Println("Error while creating proxy request - ", err)
		return nil, "", 500
	}
	defer r.Body.Close()

	// Copy headers to new request
	for header, values := range r.Header {
		if rf.hopByHopHeaders[header] { // skip hop-by-hop headers
			continue
		}
		for _, value := range values {
			proxyRequest.Header.Add(header, value)
		}
	}

	// Add proxy-specific headers
	proxyRequest.Header.Set("X-Forwarded-For", r.RemoteAddr)
	proxyRequest.Header.Set("X-Forwarded-Host", r.Host)
	proxyRequest.Header.Set("X-Forwarded-Proto", r.Proto)

	return proxyRequest, backendURL, 0
}

// Implements Request Forwarding
func (rf *RequestForwarder) RequestHandler(w http.ResponseWriter, r *http.Request) {
	// Remove port from ip
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}

	// Check if rate limit has been exhausted
	if rf.rl != nil && !rf.rl.Allow(host) {
		log.Println("Rate Limit exhausted")
		WriteResponse(w, 429, "Too Many Requests")
		return
	}

	// Check if cache has the resource
	var cacheResult cache.LookupResult
	if rf.c != nil {
		cacheResult = rf.c.Lookup(r)
		if cacheResult.Action == cache.CacheHit {
			// Copy headers to response
			for key, values := range cacheResult.Resource.Header {
				for _, value := range values {
					w.Header().Add(key, value)
				}
			}

			// Add age header and status code
			w.Header().Add("age", cacheResult.Age)
			w.WriteHeader(200)

			w.Write(cacheResult.Resource.Body)
			return
		}
	}

	// Construct Proxy Request
	proxyRequest, backendURL, errorCode := rf.constructRequest(r)

	if errorCode == 503 {
		WriteResponse(w, 503, "Service Unavailable")
		return
	} else if errorCode == 500 {
		WriteResponse(w, 500, "Internal Server Error")
		return
	}
	defer rf.lb.Finished(backendURL)

	// Add If-None-Match header for Etag revalidation
	if rf.c != nil && cacheResult.Action == cache.CacheMustRevalidate {
		proxyRequest.Header.Add("If-None-Match", cacheResult.ETag)
	}

	// Send request to backend server
	resp, err := rf.httpClient.Do(proxyRequest)
	if err != nil {
		rf.sp.MarkError(backendURL) // mark server error
		log.Println("Backend service returned an error - ", err)
		WriteResponse(w, 502, "Bad Gateway")
		return
	}
	defer resp.Body.Close()

	// Cache says - Must revalidate with backend
	// If Nothing changed, return cached response
	if rf.c != nil && cacheResult.Action == cache.CacheMustRevalidate && resp.StatusCode == 304 {
		rf.c.Refresh(r)
		// Copy headers to response
		for key, values := range cacheResult.Resource.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}

		// Add age header and status code
		w.Header().Add("age", "0")
		w.WriteHeader(200)

		w.Write(cacheResult.Resource.Body)
		return
	}

	// Cache response
	if rf.c != nil && (r.Method == "GET" || r.Method == "HEAD") {
		rf.c.Put(r, resp)
	}

	// Copy response headers back to user
	for header, values := range resp.Header {
		if rf.hopByHopHeaders[header] { // skip hop-by-hop headers
			continue
		}
		for _, value := range values {
			w.Header().Add(header, value)
		}
	}

	// Stream response back to user
	w.WriteHeader(resp.StatusCode)
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		log.Println("Error streaming response to user - ", err)
	}
}

// Helper function to write responses to user
func WriteResponse(w http.ResponseWriter, statusCode int, body string) {
	w.Header().Set("Content-type", "Application/json")
	w.WriteHeader(statusCode)

	resp := make(map[string]string)
	resp["message"] = body
	json.NewEncoder(w).Encode(resp)
}
