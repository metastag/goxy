package main

// make http server port a configurable option in yaml file

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
)

var (
	httpClient      = http.Client{Timeout: 20 * time.Second}
	hopByHopHeaders = map[string]bool{
		"Connection":          true,
		"Keep-Alive":          true,
		"Proxy-Authenticate":  true,
		"Proxy-Authorization": true,
		"TE":                  true,
		"Trailers":            true,
		"Transfer-Encoding":   true,
		"Upgrade":             true,
	}
)

// Helper function to write responses to user
func WriteResponse(w http.ResponseWriter, statusCode int, body string) {
	w.Header().Set("Content-type", "Application/json")
	w.WriteHeader(statusCode)

	resp := make(map[string]string)
	resp["message"] = body
	json.NewEncoder(w).Encode(resp)
}

// Manages Request Forwarding
func requestHandler(w http.ResponseWriter, r *http.Request) {
	// Construct proxy request

	PATH := "http://localhost:8081" // Fetch PATH from config file (TODO)
	forwardUrl := PATH + r.URL.Path

	// Add any query parameters to path
	if r.URL.RawQuery != "" {
		forwardUrl += "?" + r.URL.RawQuery
	}

	// Generate new request with same method and body
	proxyRequest, err := http.NewRequest(r.Method, forwardUrl, r.Body)
	if err != nil {
		log.Printf("Error creating proxy request - %s\n", err)
		WriteResponse(w, 500, "Internal Server Error")
		return
	}
	defer r.Body.Close()

	// Copy headers to new request
	for header, values := range r.Header {
		if hopByHopHeaders[header] { // skip hop-by-hop headers
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

	resp, err := httpClient.Do(proxyRequest)
	if err != nil {
		log.Printf("Backend service returned an error - %s\n", err)
		WriteResponse(w, 502, "Bad Gateway")
		return
	}
	defer resp.Body.Close()

	// Copy response headers back to user
	for header, values := range resp.Header {
		if hopByHopHeaders[header] { // skip hop-by-hop headers
			continue
		}
		for _, value := range values {
			w.Header().Add(header, value)
		}
	}

	// Stream response back to client
	w.WriteHeader(resp.StatusCode)
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		log.Printf("Error streaming response to user - %s\n", err)
	}
}

func main() {
	// Fetch PORT from config file (TODO)
	PORT := ":" + "8080"

	// Start the server on PORT
	log.Println("Server starting on port", PORT)
	err := http.ListenAndServe(PORT, http.HandlerFunc(requestHandler)) // one handler for any request
	if err != nil {
		log.Println("Error starting the server - ", err)
	}
}
