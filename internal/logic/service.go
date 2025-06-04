package logic

import (
	"errors"
	"math"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

// Backend represents a single backend server.
type Backend struct {
	URL    *url.URL
	Proxy  *httputil.ReverseProxy
	active int32
}

// LoadBalancer distributes requests using a k+ balancing algorithm.
type LoadBalancer struct {
	backends []*Backend
	k        int
	mu       sync.RWMutex
}

// NewLoadBalancer constructs a LoadBalancer from backend URLs.
func NewLoadBalancer(k int, rawURLs []string) (*LoadBalancer, error) {
	if k <= 0 {
		return nil, errors.New("k must be positive")
	}
	lb := &LoadBalancer{k: k}
	for _, raw := range rawURLs {
		u, err := url.Parse(raw)
		if err != nil {
			return nil, err
		}
		b := &Backend{
			URL:   u,
			Proxy: httputil.NewSingleHostReverseProxy(u),
		}
		lb.backends = append(lb.backends, b)
	}
	if len(lb.backends) == 0 {
		return nil, errors.New("no backends provided")
	}
	rand.Seed(time.Now().UnixNano())
	return lb, nil
}

// Backends returns a copy of backend slice.
func (lb *LoadBalancer) Backends() []*Backend {
	lb.mu.RLock()
	defer lb.mu.RUnlock()
	b := make([]*Backend, len(lb.backends))
	copy(b, lb.backends)
	return b
}

// ServeHTTP satisfies http.Handler.
func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	backend, err := lb.chooseBackend()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	atomic.AddInt32(&backend.active, 1)
	defer atomic.AddInt32(&backend.active, -1)
	backend.Proxy.ServeHTTP(w, r)
}

func (lb *LoadBalancer) chooseBackend() (*Backend, error) {
	lb.mu.RLock()
	defer lb.mu.RUnlock()
	n := len(lb.backends)
	if n == 0 {
		return nil, errors.New("no backends available")
	}
	k := lb.k
	if k > n {
		k = n
	}
	indices := rand.Perm(n)[:k]
	var selected *Backend
	min := int32(math.MaxInt32)
	for _, idx := range indices {
		b := lb.backends[idx]
		c := atomic.LoadInt32(&b.active)
		if c < min {
			min = c
			selected = b
		}
	}
	return selected, nil
}
