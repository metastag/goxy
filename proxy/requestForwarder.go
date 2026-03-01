package proxy

import (
	"encoding/json"
	"goxy/loadbalancer"
	"io"
	"log"
	"net/http"
	"time"
)

// Represents a Request Forwarding system
type RequestForwarder struct {
	lb              loadbalancer.LoadBalancer
	httpClient      http.Client
	hopByHopHeaders map[string]bool
}

// Initialize a new Request Forwarder
func NewRequestForwarder(lb loadbalancer.LoadBalancer) *RequestForwarder {
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
	requestForwarder := RequestForwarder{lb: lb, httpClient: httpClient, hopByHopHeaders: hopByHopHeaders}
	return &requestForwarder
}

// Implements Request Forwarding
func (rf *RequestForwarder) RequestHandler(w http.ResponseWriter, r *http.Request) {

	// Construct proxy url
	path, err := rf.lb.GetNext(r.Host + r.URL.String()) // Load balancer assigns a backend server
	if err != nil {
		log.Println(err)
		WriteResponse(w, 503, "Service Unavailable")
		return
	}
	defer rf.lb.Finished(r.Host + r.URL.String())

	forwardUrl := path + r.URL.Path

	// Add any query parameters to path
	if r.URL.RawQuery != "" {
		forwardUrl += r.URL.String()
	}

	// Generate new request with same method and body
	proxyRequest, err := http.NewRequest(r.Method, forwardUrl, r.Body)
	if err != nil {
		log.Println("Error creating proxy request - ", err)
		WriteResponse(w, 500, "Internal Server Error")
		return
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

	// Send request to backend server
	resp, err := rf.httpClient.Do(proxyRequest)
	if err != nil {
		log.Println("Backend service returned an error - ", err)
		WriteResponse(w, 502, "Bad Gateway")
		return
	}
	defer resp.Body.Close()

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
