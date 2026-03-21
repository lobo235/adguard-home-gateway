package api_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lobo235/adguard-home-gateway/internal/adguard"
	"github.com/lobo235/adguard-home-gateway/internal/api"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

const testAPIKey = "test-api-key"
const testVersion = "v1.0.0-test"

// mockAdguard is a configurable mock that satisfies the adguardClient interface.
type mockAdguard struct {
	pingFunc          func() error
	listRewritesFunc  func() ([]adguard.Rewrite, error)
	addRewriteFunc    func(domain, answer string) error
	deleteRewriteFunc func(domain, answer string) error
}

func (m *mockAdguard) Ping() error {
	if m.pingFunc != nil {
		return m.pingFunc()
	}
	return nil
}

func (m *mockAdguard) ListRewrites() ([]adguard.Rewrite, error) {
	if m.listRewritesFunc != nil {
		return m.listRewritesFunc()
	}
	return nil, nil
}

func (m *mockAdguard) AddRewrite(domain, answer string) error {
	if m.addRewriteFunc != nil {
		return m.addRewriteFunc(domain, answer)
	}
	return nil
}

func (m *mockAdguard) DeleteRewrite(domain, answer string) error {
	if m.deleteRewriteFunc != nil {
		return m.deleteRewriteFunc(domain, answer)
	}
	return nil
}

// newTestServer creates a test HTTP server with the given mock adguard client.
func newTestServer(t *testing.T, mock *mockAdguard) *httptest.Server {
	t.Helper()
	srv := api.NewServer(mock, testAPIKey, testVersion, discardLogger())
	return httptest.NewServer(srv.Handler())
}

func authHeader() string {
	return "Bearer " + testAPIKey
}

// --- helpers ---

func getJSON(t *testing.T, srv *httptest.Server, path string, auth bool) *http.Response {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, srv.URL+path, nil)
	if auth {
		req.Header.Set("Authorization", authHeader())
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	return resp
}

func assertStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	if resp.StatusCode != want {
		t.Errorf("status = %d, want %d", resp.StatusCode, want)
	}
}

func assertErrorCode(t *testing.T, resp *http.Response, wantCode string) {
	t.Helper()
	var body struct {
		Code string `json:"code"`
	}
	json.NewDecoder(resp.Body).Decode(&body)
	if body.Code != wantCode {
		t.Errorf("error code = %q, want %q", body.Code, wantCode)
	}
}

// --- auth middleware ---

func TestAuth_MissingToken(t *testing.T) {
	srv := newTestServer(t, &mockAdguard{})
	defer srv.Close()
	resp := getJSON(t, srv, "/rewrites", false)
	assertStatus(t, resp, http.StatusUnauthorized)
	assertErrorCode(t, resp, "unauthorized")
}

func TestAuth_WrongToken(t *testing.T) {
	srv := newTestServer(t, &mockAdguard{})
	defer srv.Close()
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/rewrites", nil)
	req.Header.Set("Authorization", "Bearer wrong-key")
	resp, _ := http.DefaultClient.Do(req)
	assertStatus(t, resp, http.StatusUnauthorized)
}

func TestAuth_ValidToken(t *testing.T) {
	srv := newTestServer(t, &mockAdguard{
		listRewritesFunc: func() ([]adguard.Rewrite, error) { return []adguard.Rewrite{}, nil },
	})
	defer srv.Close()
	resp := getJSON(t, srv, "/rewrites", true)
	assertStatus(t, resp, http.StatusOK)
}

// --- GET /health ---

func TestHealth_AdGuardUp(t *testing.T) {
	srv := newTestServer(t, &mockAdguard{
		pingFunc: func() error { return nil },
	})
	defer srv.Close()

	resp := getJSON(t, srv, "/health", false) // no auth required
	assertStatus(t, resp, http.StatusOK)

	var body struct {
		Status  string `json:"status"`
		Version string `json:"version"`
	}
	json.NewDecoder(resp.Body).Decode(&body)
	if body.Status != "ok" {
		t.Errorf("status = %q, want ok", body.Status)
	}
	if body.Version != testVersion {
		t.Errorf("version = %q, want %q", body.Version, testVersion)
	}
}

func TestHealth_AdGuardDown(t *testing.T) {
	srv := newTestServer(t, &mockAdguard{
		pingFunc: func() error { return errors.New("connection refused") },
	})
	defer srv.Close()

	resp := getJSON(t, srv, "/health", false)
	assertStatus(t, resp, http.StatusServiceUnavailable)

	var body struct {
		Status string `json:"status"`
	}
	json.NewDecoder(resp.Body).Decode(&body)
	if body.Status != "unavailable" {
		t.Errorf("status = %q, want unavailable", body.Status)
	}
}

func TestHealth_NoAuthRequired(t *testing.T) {
	srv := newTestServer(t, &mockAdguard{})
	defer srv.Close()
	// No Authorization header — should still get a response (not 401)
	resp := getJSON(t, srv, "/health", false)
	if resp.StatusCode == http.StatusUnauthorized {
		t.Error("/health should not require auth")
	}
}

// --- GET /rewrites ---

func TestListRewrites_OK(t *testing.T) {
	want := []adguard.Rewrite{
		{Domain: "foo.example.com", Answer: "10.0.0.1"},
		{Domain: "bar.example.com", Answer: "10.0.0.2"},
	}
	srv := newTestServer(t, &mockAdguard{
		listRewritesFunc: func() ([]adguard.Rewrite, error) { return want, nil },
	})
	defer srv.Close()

	resp := getJSON(t, srv, "/rewrites", true)
	assertStatus(t, resp, http.StatusOK)

	var got []adguard.Rewrite
	json.NewDecoder(resp.Body).Decode(&got)
	if len(got) != len(want) {
		t.Fatalf("got %d rewrites, want %d", len(got), len(want))
	}
}

func TestListRewrites_UpstreamError(t *testing.T) {
	srv := newTestServer(t, &mockAdguard{
		listRewritesFunc: func() ([]adguard.Rewrite, error) {
			return nil, errors.New("adguard unavailable")
		},
	})
	defer srv.Close()
	resp := getJSON(t, srv, "/rewrites", true)
	assertStatus(t, resp, http.StatusBadGateway)
	assertErrorCode(t, resp, "upstream_error")
}

// --- POST /rewrites ---

func postJSON(t *testing.T, srv *httptest.Server, path string, body any) *http.Response {
	t.Helper()
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, srv.URL+path, bytes.NewReader(b))
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	return resp
}

func TestAddRewrite_OK(t *testing.T) {
	var gotDomain, gotAnswer string
	srv := newTestServer(t, &mockAdguard{
		addRewriteFunc: func(domain, answer string) error {
			gotDomain, gotAnswer = domain, answer
			return nil
		},
	})
	defer srv.Close()

	resp := postJSON(t, srv, "/rewrites", map[string]string{
		"domain": "svc.example.com",
		"answer": "10.0.0.5",
	})
	assertStatus(t, resp, http.StatusCreated)
	if gotDomain != "svc.example.com" || gotAnswer != "10.0.0.5" {
		t.Errorf("AddRewrite called with (%q, %q)", gotDomain, gotAnswer)
	}
}

func TestAddRewrite_MissingDomain(t *testing.T) {
	srv := newTestServer(t, &mockAdguard{})
	defer srv.Close()
	resp := postJSON(t, srv, "/rewrites", map[string]string{"answer": "10.0.0.5"})
	assertStatus(t, resp, http.StatusBadRequest)
	assertErrorCode(t, resp, "missing_fields")
}

func TestAddRewrite_MissingAnswer(t *testing.T) {
	srv := newTestServer(t, &mockAdguard{})
	defer srv.Close()
	resp := postJSON(t, srv, "/rewrites", map[string]string{"domain": "svc.example.com"})
	assertStatus(t, resp, http.StatusBadRequest)
	assertErrorCode(t, resp, "missing_fields")
}

func TestAddRewrite_InvalidJSON(t *testing.T) {
	srv := newTestServer(t, &mockAdguard{})
	defer srv.Close()
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/rewrites", bytes.NewBufferString("not json"))
	req.Header.Set("Authorization", authHeader())
	resp, _ := http.DefaultClient.Do(req)
	assertStatus(t, resp, http.StatusBadRequest)
	assertErrorCode(t, resp, "invalid_body")
}

func TestAddRewrite_UpstreamError(t *testing.T) {
	srv := newTestServer(t, &mockAdguard{
		addRewriteFunc: func(domain, answer string) error {
			return errors.New("adguard error")
		},
	})
	defer srv.Close()
	resp := postJSON(t, srv, "/rewrites", map[string]string{
		"domain": "svc.example.com",
		"answer": "10.0.0.5",
	})
	assertStatus(t, resp, http.StatusBadGateway)
	assertErrorCode(t, resp, "upstream_error")
}

// --- PUT /rewrites/{domain} ---

func putJSON(t *testing.T, srv *httptest.Server, path string, body any) *http.Response {
	t.Helper()
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPut, srv.URL+path, bytes.NewReader(b))
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PUT %s: %v", path, err)
	}
	return resp
}

func TestUpdateRewrite_OK(t *testing.T) {
	existing := []adguard.Rewrite{{Domain: "svc.example.com", Answer: "10.0.0.1"}}
	var deletedAnswer, addedAnswer string
	srv := newTestServer(t, &mockAdguard{
		listRewritesFunc:  func() ([]adguard.Rewrite, error) { return existing, nil },
		deleteRewriteFunc: func(domain, answer string) error { deletedAnswer = answer; return nil },
		addRewriteFunc:    func(domain, answer string) error { addedAnswer = answer; return nil },
	})
	defer srv.Close()

	resp := putJSON(t, srv, "/rewrites/svc.example.com", map[string]string{"answer": "10.0.0.2"})
	assertStatus(t, resp, http.StatusOK)
	if deletedAnswer != "10.0.0.1" {
		t.Errorf("deleted answer = %q, want 10.0.0.1", deletedAnswer)
	}
	if addedAnswer != "10.0.0.2" {
		t.Errorf("added answer = %q, want 10.0.0.2", addedAnswer)
	}
}

func TestUpdateRewrite_Upsert(t *testing.T) {
	// PUT on a domain with no existing rewrite should create it (upsert).
	var addedDomain, addedAnswer string
	srv := newTestServer(t, &mockAdguard{
		listRewritesFunc: func() ([]adguard.Rewrite, error) { return []adguard.Rewrite{}, nil },
		addRewriteFunc: func(domain, answer string) error {
			addedDomain, addedAnswer = domain, answer
			return nil
		},
	})
	defer srv.Close()
	resp := putJSON(t, srv, "/rewrites/new.example.com", map[string]string{"answer": "10.0.0.9"})
	assertStatus(t, resp, http.StatusOK)
	if addedDomain != "new.example.com" || addedAnswer != "10.0.0.9" {
		t.Errorf("AddRewrite called with (%q, %q)", addedDomain, addedAnswer)
	}
}

func TestUpdateRewrite_MissingAnswer(t *testing.T) {
	srv := newTestServer(t, &mockAdguard{})
	defer srv.Close()
	resp := putJSON(t, srv, "/rewrites/svc.example.com", map[string]string{})
	assertStatus(t, resp, http.StatusBadRequest)
	assertErrorCode(t, resp, "missing_fields")
}

func TestUpdateRewrite_ListUpstreamError(t *testing.T) {
	srv := newTestServer(t, &mockAdguard{
		listRewritesFunc: func() ([]adguard.Rewrite, error) {
			return nil, errors.New("adguard unavailable")
		},
	})
	defer srv.Close()
	resp := putJSON(t, srv, "/rewrites/svc.example.com", map[string]string{"answer": "10.0.0.2"})
	assertStatus(t, resp, http.StatusBadGateway)
	assertErrorCode(t, resp, "upstream_error")
}

// --- DELETE /rewrites/{domain} ---

func TestDeleteRewrite_OK(t *testing.T) {
	var gotDomain, gotAnswer string
	srv := newTestServer(t, &mockAdguard{
		listRewritesFunc: func() ([]adguard.Rewrite, error) {
			return []adguard.Rewrite{{Domain: "svc.example.com", Answer: "10.0.0.1"}}, nil
		},
		deleteRewriteFunc: func(domain, answer string) error {
			gotDomain, gotAnswer = domain, answer
			return nil
		},
	})
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/rewrites/svc.example.com", nil)
	req.Header.Set("Authorization", authHeader())
	resp, _ := http.DefaultClient.Do(req)
	assertStatus(t, resp, http.StatusNoContent)
	if gotDomain != "svc.example.com" || gotAnswer != "10.0.0.1" {
		t.Errorf("DeleteRewrite called with (%q, %q)", gotDomain, gotAnswer)
	}
}

func TestDeleteRewrite_DomainNotFound(t *testing.T) {
	srv := newTestServer(t, &mockAdguard{
		listRewritesFunc: func() ([]adguard.Rewrite, error) { return []adguard.Rewrite{}, nil },
	})
	defer srv.Close()
	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/rewrites/missing.example.com", nil)
	req.Header.Set("Authorization", authHeader())
	resp, _ := http.DefaultClient.Do(req)
	assertStatus(t, resp, http.StatusNotFound)
	assertErrorCode(t, resp, "not_found")
}

func TestDeleteRewrite_ListUpstreamError(t *testing.T) {
	srv := newTestServer(t, &mockAdguard{
		listRewritesFunc: func() ([]adguard.Rewrite, error) {
			return nil, errors.New("adguard unavailable")
		},
	})
	defer srv.Close()
	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/rewrites/svc.example.com", nil)
	req.Header.Set("Authorization", authHeader())
	resp, _ := http.DefaultClient.Do(req)
	assertStatus(t, resp, http.StatusBadGateway)
	assertErrorCode(t, resp, "upstream_error")
}

func TestDeleteRewrite_UpstreamError(t *testing.T) {
	srv := newTestServer(t, &mockAdguard{
		listRewritesFunc: func() ([]adguard.Rewrite, error) {
			return []adguard.Rewrite{{Domain: "svc.example.com", Answer: "10.0.0.1"}}, nil
		},
		deleteRewriteFunc: func(domain, answer string) error {
			return errors.New("adguard error")
		},
	})
	defer srv.Close()
	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/rewrites/svc.example.com", nil)
	req.Header.Set("Authorization", authHeader())
	resp, _ := http.DefaultClient.Do(req)
	assertStatus(t, resp, http.StatusBadGateway)
	assertErrorCode(t, resp, "upstream_error")
}

// --- GET /rewrites/{domain} ---

func TestGetRewrite_OK(t *testing.T) {
	want := adguard.Rewrite{Domain: "svc.example.com", Answer: "10.0.0.5"}
	srv := newTestServer(t, &mockAdguard{
		listRewritesFunc: func() ([]adguard.Rewrite, error) {
			return []adguard.Rewrite{
				{Domain: "other.example.com", Answer: "10.0.0.1"},
				want,
			}, nil
		},
	})
	defer srv.Close()

	resp := getJSON(t, srv, "/rewrites/svc.example.com", true)
	assertStatus(t, resp, http.StatusOK)

	var got adguard.Rewrite
	json.NewDecoder(resp.Body).Decode(&got)
	if got.Domain != want.Domain || got.Answer != want.Answer {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestGetRewrite_NotFound(t *testing.T) {
	srv := newTestServer(t, &mockAdguard{
		listRewritesFunc: func() ([]adguard.Rewrite, error) { return []adguard.Rewrite{}, nil },
	})
	defer srv.Close()
	resp := getJSON(t, srv, "/rewrites/missing.example.com", true)
	assertStatus(t, resp, http.StatusNotFound)
	assertErrorCode(t, resp, "not_found")
}

func TestGetRewrite_UpstreamError(t *testing.T) {
	srv := newTestServer(t, &mockAdguard{
		listRewritesFunc: func() ([]adguard.Rewrite, error) {
			return nil, errors.New("adguard unavailable")
		},
	})
	defer srv.Close()
	resp := getJSON(t, srv, "/rewrites/svc.example.com", true)
	assertStatus(t, resp, http.StatusBadGateway)
	assertErrorCode(t, resp, "upstream_error")
}
