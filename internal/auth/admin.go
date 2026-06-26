package auth

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// UserAdmin performs trusted Supabase Auth Admin operations.
type UserAdmin interface {
	DeleteUser(ctx context.Context, id string) error
}

// AdminClient talks to Supabase Auth Admin API using the service role key. It
// must only run on the backend; never ship the key in a public client.
type AdminClient struct {
	baseURL    string
	serviceKey string
	httpClient *http.Client
}

// NewAdminClient builds a Supabase Auth Admin client.
func NewAdminClient(baseURL, serviceKey string) *AdminClient {
	return &AdminClient{
		baseURL:    strings.TrimRight(baseURL, "/"),
		serviceKey: serviceKey,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// DeleteUser permanently deletes a Supabase Auth user. Database rows that
// reference auth.users are cleaned up by ON DELETE CASCADE constraints.
func (c *AdminClient) DeleteUser(ctx context.Context, id string) error {
	if c == nil || c.baseURL == "" || c.serviceKey == "" {
		return ErrNotConfigured
	}

	endpoint := c.baseURL + "/auth/v1/admin/users/" + url.PathEscape(id)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.serviceKey)
	req.Header.Set("apikey", c.serviceKey)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(res.Body, 2048))
		return fmt.Errorf("delete auth user: %s %s", res.Status, strings.TrimSpace(string(raw)))
	}
	return nil
}
