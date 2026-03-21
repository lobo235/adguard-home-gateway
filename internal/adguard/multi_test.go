package adguard_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lobo235/adguard-home-gateway/internal/adguard"
)

// makeServer returns a test server that responds to all AdGuard API paths with
// the given status code, plus an optional rewrite list for GET /control/rewrite/list.
func makeServer(status int, rewrites []adguard.Rewrite) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/control/rewrite/list" && rewrites != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(rewrites)
			return
		}
		w.WriteHeader(status)
	}))
}

func clientFor(srv *httptest.Server) *adguard.Client {
	return adguard.NewClient(srv.URL, "", "", false)
}

func multiClientFor(srvs ...*httptest.Server) (*adguard.MultiClient, []string) {
	addrs := make([]string, len(srvs))
	clients := make([]*adguard.Client, len(srvs))
	for i, srv := range srvs {
		addrs[i] = srv.URL
		clients[i] = clientFor(srv)
	}
	return adguard.NewMultiClient(addrs, clients), addrs
}

// --- Ping ---

func TestMultiPing_AllOK(t *testing.T) {
	s1 := makeServer(http.StatusOK, nil)
	s2 := makeServer(http.StatusOK, nil)
	defer s1.Close()
	defer s2.Close()

	mc, _ := multiClientFor(s1, s2)
	if err := mc.Ping(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestMultiPing_OneDown(t *testing.T) {
	s1 := makeServer(http.StatusOK, nil)
	s2 := makeServer(http.StatusServiceUnavailable, nil)
	defer s1.Close()
	defer s2.Close()

	mc, _ := multiClientFor(s1, s2)
	if err := mc.Ping(); err == nil {
		t.Fatal("expected error when one server is down")
	}
}

func TestMultiPing_AllDown(t *testing.T) {
	s1 := makeServer(http.StatusServiceUnavailable, nil)
	s2 := makeServer(http.StatusServiceUnavailable, nil)
	defer s1.Close()
	defer s2.Close()

	mc, _ := multiClientFor(s1, s2)
	err := mc.Ping()
	if err == nil {
		t.Fatal("expected error when all servers are down")
	}
	if !strings.Contains(err.Error(), "2/2") {
		t.Errorf("error should report 2/2 unreachable, got: %v", err)
	}
}

// --- ListRewrites ---

func TestMultiListRewrites_FirstServerOK(t *testing.T) {
	want := []adguard.Rewrite{{Domain: "svc.example.com", Answer: "10.0.0.1"}}
	s1 := makeServer(http.StatusOK, want)
	s2 := makeServer(http.StatusOK, nil) // s2 should not be hit
	defer s1.Close()
	defer s2.Close()

	mc, _ := multiClientFor(s1, s2)
	got, err := mc.ListRewrites()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(got) != 1 || got[0].Domain != want[0].Domain {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestMultiListRewrites_FallsBackToSecondServer(t *testing.T) {
	want := []adguard.Rewrite{{Domain: "svc.example.com", Answer: "10.0.0.1"}}
	s1 := makeServer(http.StatusInternalServerError, nil) // first server fails
	s2 := makeServer(http.StatusOK, want)
	defer s1.Close()
	defer s2.Close()

	mc, _ := multiClientFor(s1, s2)
	got, err := mc.ListRewrites()
	if err != nil {
		t.Fatalf("expected fallback to succeed, got %v", err)
	}
	if len(got) != 1 {
		t.Errorf("got %d rewrites, want 1", len(got))
	}
}

func TestMultiListRewrites_AllFail(t *testing.T) {
	s1 := makeServer(http.StatusInternalServerError, nil)
	s2 := makeServer(http.StatusInternalServerError, nil)
	defer s1.Close()
	defer s2.Close()

	mc, _ := multiClientFor(s1, s2)
	if _, err := mc.ListRewrites(); err == nil {
		t.Fatal("expected error when all servers fail")
	}
}

// --- AddRewrite ---

func TestMultiAddRewrite_AllOK(t *testing.T) {
	hits := make([]int, 2)
	handlers := []http.HandlerFunc{
		func(w http.ResponseWriter, r *http.Request) { hits[0]++; w.WriteHeader(http.StatusOK) },
		func(w http.ResponseWriter, r *http.Request) { hits[1]++; w.WriteHeader(http.StatusOK) },
	}
	s1 := httptest.NewServer(handlers[0])
	s2 := httptest.NewServer(handlers[1])
	defer s1.Close()
	defer s2.Close()

	mc, _ := multiClientFor(s1, s2)
	if err := mc.AddRewrite("svc.example.com", "10.0.0.1"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if hits[0] == 0 || hits[1] == 0 {
		t.Errorf("expected both servers to be hit, got hits=%v", hits)
	}
}

func TestMultiAddRewrite_PartialFailure(t *testing.T) {
	s1 := makeServer(http.StatusOK, nil)
	s2 := makeServer(http.StatusInternalServerError, nil)
	defer s1.Close()
	defer s2.Close()

	mc, _ := multiClientFor(s1, s2)
	err := mc.AddRewrite("svc.example.com", "10.0.0.1")
	if err == nil {
		t.Fatal("expected error on partial failure")
	}
	if !strings.Contains(err.Error(), "1/2") {
		t.Errorf("error should report 1/2 failed, got: %v", err)
	}
}

func TestMultiAddRewrite_AllFail(t *testing.T) {
	s1 := makeServer(http.StatusInternalServerError, nil)
	s2 := makeServer(http.StatusInternalServerError, nil)
	defer s1.Close()
	defer s2.Close()

	mc, _ := multiClientFor(s1, s2)
	err := mc.AddRewrite("svc.example.com", "10.0.0.1")
	if err == nil {
		t.Fatal("expected error when all servers fail")
	}
	if !strings.Contains(err.Error(), "2/2") {
		t.Errorf("error should report 2/2 failed, got: %v", err)
	}
}

// --- DeleteRewrite ---

func TestMultiDeleteRewrite_AllOK(t *testing.T) {
	s1 := makeServer(http.StatusOK, nil)
	s2 := makeServer(http.StatusOK, nil)
	defer s1.Close()
	defer s2.Close()

	mc, _ := multiClientFor(s1, s2)
	if err := mc.DeleteRewrite("svc.example.com", "10.0.0.1"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestMultiDeleteRewrite_PartialFailure(t *testing.T) {
	s1 := makeServer(http.StatusOK, nil)
	s2 := makeServer(http.StatusInternalServerError, nil)
	defer s1.Close()
	defer s2.Close()

	mc, _ := multiClientFor(s1, s2)
	err := mc.DeleteRewrite("svc.example.com", "10.0.0.1")
	if err == nil {
		t.Fatal("expected error on partial failure")
	}
}

func TestMultiFanOut_AllServersAlwaysAttempted(t *testing.T) {
	// Even if the first server fails, the second must still be called.
	hits := make([]int, 2)
	s1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits[0]++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	s2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits[1]++
		w.WriteHeader(http.StatusOK)
	}))
	defer s1.Close()
	defer s2.Close()

	mc, _ := multiClientFor(s1, s2)
	mc.AddRewrite("svc.example.com", "10.0.0.1") // error expected, but we're testing side effects
	if hits[0] == 0 || hits[1] == 0 {
		t.Errorf("expected both servers to be attempted, got hits=%v", hits)
	}
}
