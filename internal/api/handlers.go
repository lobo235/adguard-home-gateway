package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// listRewritesHandler handles GET /rewrites.
func (s *Server) listRewritesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rewrites, err := s.adguard.ListRewrites()
		if err != nil {
			s.log.Error("failed to list rewrites", "error", err)
			writeError(w, http.StatusBadGateway, "upstream_error", "failed to list rewrites from AdGuard")
			return
		}
		writeJSON(w, http.StatusOK, rewrites)
	}
}

// addRewriteHandler handles POST /rewrites.
func (s *Server) addRewriteHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Domain string `json:"domain"`
			Answer string `json:"answer"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body", "request body must be valid JSON with domain and answer fields")
			return
		}
		if body.Domain == "" || body.Answer == "" {
			writeError(w, http.StatusBadRequest, "missing_fields", "domain and answer are required")
			return
		}

		if err := s.adguard.AddRewrite(body.Domain, body.Answer); err != nil {
			s.log.Error("failed to add rewrite", "domain", body.Domain, "error", err)
			writeError(w, http.StatusBadGateway, "upstream_error", "failed to add rewrite")
			return
		}
		writeJSON(w, http.StatusCreated, map[string]string{"domain": body.Domain, "answer": body.Answer})
	}
}

// updateRewriteHandler handles PUT /rewrites/{domain}.
// Looks up the existing rewrite by domain (assumes one rewrite per domain),
// deletes it, then adds a new one with the provided answer.
func (s *Server) updateRewriteHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		domain := r.PathValue("domain")

		var body struct {
			Answer string `json:"answer"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body", "request body must be valid JSON with an answer field")
			return
		}
		if body.Answer == "" {
			writeError(w, http.StatusBadRequest, "missing_fields", "answer is required")
			return
		}

		rewrites, err := s.adguard.ListRewrites()
		if err != nil {
			s.log.Error("failed to list rewrites for update", "domain", domain, "error", err)
			writeError(w, http.StatusBadGateway, "upstream_error", "failed to fetch existing rewrites")
			return
		}

		var oldAnswer string
		for _, rw := range rewrites {
			if rw.Domain == domain {
				oldAnswer = rw.Answer
				break
			}
		}
		if oldAnswer == "" {
			writeError(w, http.StatusNotFound, "not_found", fmt.Sprintf("no rewrite found for domain %q", domain))
			return
		}

		if err := s.adguard.DeleteRewrite(domain, oldAnswer); err != nil {
			s.log.Error("failed to delete old rewrite during update", "domain", domain, "error", err)
			writeError(w, http.StatusBadGateway, "upstream_error", "failed to update rewrite")
			return
		}
		if err := s.adguard.AddRewrite(domain, body.Answer); err != nil {
			s.log.Error("failed to add new rewrite during update", "domain", domain, "error", err)
			writeError(w, http.StatusBadGateway, "upstream_error", "failed to update rewrite")
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"domain": domain, "answer": body.Answer})
	}
}

// deleteRewriteHandler handles DELETE /rewrites/{domain}.
// Requires the ?answer= query parameter to identify the exact rewrite entry.
func (s *Server) deleteRewriteHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		domain := r.PathValue("domain")
		answer := r.URL.Query().Get("answer")
		if answer == "" {
			writeError(w, http.StatusBadRequest, "missing_param", "answer query parameter is required")
			return
		}

		if err := s.adguard.DeleteRewrite(domain, answer); err != nil {
			s.log.Error("failed to delete rewrite", "domain", domain, "error", err)
			writeError(w, http.StatusBadGateway, "upstream_error", "failed to delete rewrite")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
