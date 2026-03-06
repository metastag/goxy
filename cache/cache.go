package cache

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Resource struct {
	Header  http.Header
	Body    []byte
	Created time.Time
}

type CacheAction int

const (
	CacheHit            CacheAction = iota // Fresh, serve directly
	CacheMiss                              // Cache cannot serve, forward to backend
	CacheMustRevalidate                    // Stale or no-cache, revalidate with backend
)

// Result of a cache lookup
type LookupResult struct {
	Action   CacheAction
	Resource *Resource
	Age      string
	ETag     string
}

// Represents a Cache system
type Cache struct {
	store    map[string]*Resource
	varyKeys map[string][]string // base key -> sorted Vary field names
	mu       sync.RWMutex
}

// Returns a new Cache struct
func NewCache() *Cache {
	store := make(map[string]*Resource)
	varyKeys := make(map[string][]string)
	cache := Cache{store: store, varyKeys: varyKeys, mu: sync.RWMutex{}}

	// Launch TTL cleanup goroutine, runs every 60 minutes
	go cache.cleanUp(60)
	return &cache
}

// Periodically removes Resources whose max age has been crossed by over 1 hour
func (c *Cache) cleanUp(duration int) {
	period := time.Duration(duration) * time.Minute
	ticker := time.NewTicker(period)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()

		for key, resource := range c.store {
			cacheHeader := CacheControlParser(resource.Header.Get("Cache-Control"))
			// Check freshness of resource
			var maxage int
			if val, ok := cacheHeader["s-maxage"]; ok {
				maxage = val
			} else if val, ok := cacheHeader["max-age"]; ok {
				maxage = val
			}

			// Current age of resource + 1 hour
			age := int(time.Since(resource.Created).Seconds() + 3600)

			if age > maxage { // Resource is stale + not accessed for 1 hour
				delete(c.store, key)
			}
		}

		c.mu.Unlock()
	}
}

// Helper function to parse Cache-Control Header
func CacheControlParser(cacheHeader string) map[string]int {
	directives := make(map[string]int)

	// Case-insensitive
	cacheHeader = strings.ToLower(cacheHeader)

	// Turn comma-seperated string into iterator
	parts := strings.SplitSeq(cacheHeader, ",")
	for part := range parts {
		part = strings.TrimSpace(part) // remove any whitespace

		// Check if part contains `=` sign
		before, after, found := strings.Cut(part, "=")
		if found {
			before := strings.TrimSpace(before)
			after, err := strconv.Atoi(after)
			if err != nil {
				continue
			}
			directives[before] = after
		} else {
			directives[part] = 1
		}
	}

	return directives
}

// Parses a Vary header into a sorted list of canonical header names
func parseVaryHeader(vary string) []string {
	if vary == "" {
		return nil
	}
	var fields []string
	for _, f := range strings.Split(vary, ",") {
		f = strings.TrimSpace(f)
		if f != "" {
			fields = append(fields, http.CanonicalHeaderKey(f))
		}
	}
	sort.Strings(fields)
	return fields
}

// Builds a full cache key by appending Vary header values from the request
func buildVaryKey(baseKey string, varyFields []string, r *http.Request) string {
	if len(varyFields) == 0 {
		return baseKey
	}
	var b strings.Builder
	b.WriteString(baseKey)
	for _, field := range varyFields {
		b.WriteString("|")
		b.WriteString(field)
		b.WriteString("=")
		b.WriteString(r.Header.Get(field))
	}
	return b.String()
}

// Checks resource in Cache
func (c *Cache) Lookup(r *http.Request) LookupResult {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Construct key to query the cache store
	baseKey := fmt.Sprintf("%s|%s|%s", r.Method, r.Host, r.URL.RequestURI())
	varyFields := c.varyKeys[baseKey]
	key := buildVaryKey(baseKey, varyFields, r)

	// Fetch the resource from cache
	resource, exists := c.store[key]
	if !exists {
		return LookupResult{Action: CacheMiss}
	}

	// Retrieve Cache-Control headers
	cacheHeader := CacheControlParser(resource.Header.Get("Cache-Control"))

	// If no-cache, revalidate with backend
	if cacheHeader["no-cache"] == 1 {
		return LookupResult{
			Action:   CacheMustRevalidate,
			Resource: resource,
			ETag:     resource.Header.Get("ETag"),
		}
	}

	// Check freshness of resource
	var maxage int
	if val, ok := cacheHeader["s-maxage"]; ok {
		maxage = val
	} else if val, ok := cacheHeader["max-age"]; ok {
		maxage = val
	}
	// Current age of resource
	age := int(time.Since(resource.Created).Seconds())

	if maxage > age { // Resource is fresh, cache hit
		return LookupResult{
			Action:   CacheHit,
			Resource: resource,
			Age:      fmt.Sprint(age),
		}
	}

	// Resource is stale, must revalidate
	return LookupResult{
		Action:   CacheMustRevalidate,
		Resource: resource,
		ETag:     resource.Header.Get("ETag"),
	}
}

// Puts a resource in Cache
func (c *Cache) Put(req *http.Request, resp *http.Response) {
	cacheHeader := CacheControlParser(resp.Header.Get("Cache-Control"))

	// Do not cache if Cache-Control=private/no-store
	if cacheHeader["private"] == 1 || cacheHeader["no-store"] == 1 {
		return
	}

	// Do not cache if Vary: * (response varies on everything)
	vary := resp.Header.Get("Vary")
	if strings.TrimSpace(vary) == "*" {
		return
	}

	baseKey := fmt.Sprintf("%s|%s|%s", req.Method, req.Host, req.URL.RequestURI())
	varyFields := parseVaryHeader(vary)
	key := buildVaryKey(baseKey, varyFields, req)

	// Extract info from response before acquiring lock
	header := resp.Header.Clone()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("Cache - Failed to extract body from response - ", err)
		return
	}
	resp.Body.Close()

	// Replace the body so it can be returned to the user
	resp.Body = io.NopCloser(bytes.NewReader(body))

	resource := Resource{
		Header:  header,
		Body:    body,
		Created: time.Now(),
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.varyKeys[baseKey] = varyFields
	c.store[key] = &resource
}

// Refreshes the freshness timer of a cache entry (called after http 304)
func (c *Cache) Refresh(req *http.Request) {
	c.mu.Lock()
	defer c.mu.Unlock()

	baseKey := fmt.Sprintf("%s|%s|%s", req.Method, req.Host, req.URL.RequestURI())
	varyFields := c.varyKeys[baseKey]
	key := buildVaryKey(baseKey, varyFields, req)
	if resource, exists := c.store[key]; exists {
		resource.Created = time.Now()
	}
}
