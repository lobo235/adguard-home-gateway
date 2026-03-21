package adguard_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lobo235/adguard-home-gateway/internal/adguard"
)

const testUser = "admin"
const testPass = "secret"

func newTestClient(srv *httptest.Server) *adguard.Client {
	return adguard.NewClient(srv.URL, testUser, testPass, false)
}

func newTestClientNoAuth(srv *httptest.Server) *adguard.Client {
	return adguard.NewClient(srv.URL, "", "", false)
}

func checkBasicAuth(t *testing.T, r *http.Request) {
	t.Helper()
	u, p, ok := r.BasicAuth()
	if !ok || u != testUser || p != testPass {
		t.Errorf("expected basic auth %s/%s, got %s/%s (ok=%v)", testUser, testPass, u, p, ok)
	}
}

func TestPing_OK(t *testing.T) {
	hit := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		checkBasicAuth(t, r)
		if r.Method != http.MethodGet || r.URL.Path != "/control/status" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		hit = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	if err := newTestClient(srv).Ping(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !hit {
		t.Fatal("mock endpoint was not called")
	}
}

func TestPing_NoAuth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, _, ok := r.BasicAuth(); ok {
			t.Error("expected no Authorization header when credentials are empty")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	if err := newTestClientNoAuth(srv).Ping(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestPing_Unavailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	if err := newTestClient(srv).Ping(); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestListRewrites_OK(t *testing.T) {
	want := []adguard.Rewrite{
		{Domain: "foo.example.com", Answer: "192.168.1.10"},
		{Domain: "bar.example.com", Answer: "192.168.1.11"},
	}

	hit := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		checkBasicAuth(t, r)
		if r.Method != http.MethodGet || r.URL.Path != "/control/rewrite/list" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		hit = true
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	got, err := newTestClient(srv).ListRewrites()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !hit {
		t.Fatal("mock endpoint was not called")
	}
	if len(got) != len(want) {
		t.Fatalf("expected %d rewrites, got %d", len(want), len(got))
	}
	for i, r := range got {
		if r.Domain != want[i].Domain || r.Answer != want[i].Answer {
			t.Errorf("rewrite[%d]: got {%s %s}, want {%s %s}", i, r.Domain, r.Answer, want[i].Domain, want[i].Answer)
		}
	}
}

func TestListRewrites_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
	}))
	defer srv.Close()

	got, err := newTestClient(srv).ListRewrites()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 rewrites, got %d", len(got))
	}
}

func TestAddRewrite_OK(t *testing.T) {
	hit := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		checkBasicAuth(t, r)
		if r.Method != http.MethodPost || r.URL.Path != "/control/rewrite/add" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body adguard.Rewrite
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding body: %v", err)
		}
		if body.Domain != "svc.example.com" || body.Answer != "10.0.0.5" {
			t.Errorf("unexpected body: %+v", body)
		}
		hit = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	if err := newTestClient(srv).AddRewrite("svc.example.com", "10.0.0.5"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !hit {
		t.Fatal("mock endpoint was not called")
	}
}

func TestAddRewrite_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	if err := newTestClient(srv).AddRewrite("svc.example.com", "10.0.0.5"); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDeleteRewrite_OK(t *testing.T) {
	hit := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		checkBasicAuth(t, r)
		if r.Method != http.MethodPost || r.URL.Path != "/control/rewrite/delete" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body adguard.Rewrite
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding body: %v", err)
		}
		if body.Domain != "svc.example.com" || body.Answer != "10.0.0.5" {
			t.Errorf("unexpected body: %+v", body)
		}
		hit = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	if err := newTestClient(srv).DeleteRewrite("svc.example.com", "10.0.0.5"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !hit {
		t.Fatal("mock endpoint was not called")
	}
}

func TestDeleteRewrite_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	if err := newTestClient(srv).DeleteRewrite("svc.example.com", "10.0.0.5"); err == nil {
		t.Fatal("expected error, got nil")
	}
}
