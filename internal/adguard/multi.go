package adguard

import (
	"fmt"
	"strings"
)

// MultiClient fans write operations out to all configured AdGuard Home servers
// and reads from the first reachable server. All servers are assumed to be
// identical replicas.
type MultiClient struct {
	addrs   []string
	clients []*Client
}

// NewMultiClient creates a MultiClient from parallel slices of addresses and
// pre-constructed Clients. The caller is responsible for ensuring addrs[i]
// corresponds to clients[i].
func NewMultiClient(addrs []string, clients []*Client) *MultiClient {
	return &MultiClient{addrs: addrs, clients: clients}
}

// Ping checks connectivity to every server. Returns an error if any server is
// unreachable so the health endpoint accurately reflects full replica availability.
func (m *MultiClient) Ping() error {
	var errs []string
	for i, c := range m.clients {
		if err := c.Ping(); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", m.addrs[i], err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%d/%d servers unreachable: %s",
			len(errs), len(m.clients), strings.Join(errs, "; "))
	}
	return nil
}

// ListRewrites returns the rewrite list from the first server that responds
// successfully. Since all servers are replicas, this is equivalent to reading
// from any server.
func (m *MultiClient) ListRewrites() ([]Rewrite, error) {
	var lastErr error
	for i, c := range m.clients {
		rewrites, err := c.ListRewrites()
		if err == nil {
			return rewrites, nil
		}
		lastErr = fmt.Errorf("%s: %w", m.addrs[i], err)
	}
	return nil, fmt.Errorf("all servers failed: %w", lastErr)
}

// AddRewrite adds a DNS rewrite entry on every server.
// Returns an error listing all servers that failed.
func (m *MultiClient) AddRewrite(domain, answer string) error {
	return m.fanOut(func(c *Client) error {
		return c.AddRewrite(domain, answer)
	})
}

// DeleteRewrite removes a DNS rewrite entry from every server.
// Returns an error listing all servers that failed.
func (m *MultiClient) DeleteRewrite(domain, answer string) error {
	return m.fanOut(func(c *Client) error {
		return c.DeleteRewrite(domain, answer)
	})
}

// fanOut calls fn against every client, collecting errors. All clients are
// always attempted regardless of earlier failures.
func (m *MultiClient) fanOut(fn func(*Client) error) error {
	var errs []string
	for i, c := range m.clients {
		if err := fn(c); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", m.addrs[i], err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("failed on %d/%d servers: %s",
			len(errs), len(m.clients), strings.Join(errs, "; "))
	}
	return nil
}
