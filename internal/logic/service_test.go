package logic

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// helper to perform a single request to lb
func makeRequest(lb http.Handler) *http.Response {
	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	rec := httptest.NewRecorder()
	lb.ServeHTTP(rec, req)
	return rec.Result()
}

func TestNewLoadBalancerErrors(t *testing.T) {
	if _, err := NewLoadBalancer(0, []string{"http://example.com"}); err == nil {
		t.Error("expected error for k <= 0")
	}
	if _, err := NewLoadBalancer(1, []string{}); err == nil {
		t.Error("expected error for no backends")
	}
	if _, err := NewLoadBalancer(1, []string{"://bad"}); err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestActiveConnections(t *testing.T) {
	done := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-done
	}))
	defer server.Close()

	lb, err := NewLoadBalancer(1, []string{server.URL})
	if err != nil {
		t.Fatal(err)
	}
	backend := lb.Backends()[0]

	go makeRequest(lb)
	time.Sleep(50 * time.Millisecond)
	if c := atomic.LoadInt32(&backend.active); c != 1 {
		t.Fatalf("expected active=1 during request, got %d", c)
	}
	close(done)
	time.Sleep(50 * time.Millisecond)
	if c := atomic.LoadInt32(&backend.active); c != 0 {
		t.Fatalf("expected active=0 after request, got %d", c)
	}
}

func TestLoadBalancerDistribution(t *testing.T) {
	const backendCount = 3
	const requests = 300
	var counts [backendCount]int32

	servers := make([]*httptest.Server, backendCount)
	for i := range servers {
		idx := i
		servers[i] = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&counts[idx], 1)
			w.WriteHeader(http.StatusOK)
		}))
		defer servers[i].Close()
	}

	urls := make([]string, backendCount)
	for i, s := range servers {
		urls[i] = s.URL
	}

	lb, err := NewLoadBalancer(2, urls)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < requests; i++ {
		resp := makeRequest(lb)
		resp.Body.Close()
	}

	metrics := make([]int32, backendCount)
	for i := range counts {
		metrics[i] = atomic.LoadInt32(&counts[i])
	}
	t.Logf("request distribution: %v", metrics)

	expected := requests / backendCount
	tolerance := expected / 2 // 50% tolerance
	for i, c := range metrics {
		diff := int(c) - expected
		if diff < -tolerance || diff > tolerance {
			t.Errorf("backend %d served %d requests, want around %d", i, c, expected)
		}
	}
}
