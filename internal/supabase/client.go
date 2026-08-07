package supabase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client talks to Supabase via postgREST (read + write) and Storage.
type Client struct {
	baseURL  string // e.g. https://<ref>.supabase.co/rest/v1
	storage  string // e.g. https://<ref>.supabase.co/storage/v1
	key      string // service_role secret
	bucket   string
	http     *http.Client
}

func New(baseURL, key, bucket string) *Client {
	rest := strings.TrimSuffix(baseURL, "/")
	storage := strings.TrimSuffix(rest, "/rest/v1") + "/storage/v1"
	return &Client{
		baseURL: rest,
		storage: storage,
		key:     key,
		bucket:  bucket,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) req(ctx context.Context, method, fullURL string, body any, prefer string) ([]byte, int, error) {
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, 0, err
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, fullURL, rd)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("apikey", c.key)
	req.Header.Set("Authorization", "Bearer "+c.key)
	req.Header.Set("Content-Type", "application/json")
	if prefer != "" {
		req.Header.Set("Prefer", prefer)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return data, resp.StatusCode, nil
}

// tableURL builds a postgREST URL for a table with optional query.
func (c *Client) tableURL(table, query string) string {
	u := c.baseURL + "/" + table
	if query != "" {
		u += "?" + query
	}
	return u
}

// FetchAll reads all rows of a table into out. opts are postgREST query params (e.g. select).
func (c *Client) FetchAll(ctx context.Context, table, selectCols string, out any) error {
	q := url.Values{}
	if selectCols != "" {
		q.Set("select", selectCols)
	}
	u := c.tableURL(table, q.Encode())
	data, status, err := c.req(ctx, http.MethodGet, u, nil, "")
	if err != nil {
		return err
	}
	if status >= 300 {
		return fmt.Errorf("supabase GET %s: status %d: %s", table, status, truncate(string(data)))
	}
	return json.Unmarshal(data, out)
}

// SelectIDs returns the set of existing ids for a table (for diff detection).
func (c *Client) SelectIDs(ctx context.Context, table string) (map[string]bool, error) {
	u := c.tableURL(table, "select=id")
	data, status, err := c.req(ctx, http.MethodGet, u, nil, "")
	if err != nil {
		return nil, err
	}
	if status >= 300 {
		return nil, fmt.Errorf("supabase select ids %s: status %d: %s", table, status, truncate(string(data)))
	}
	var rows []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(data, &rows); err != nil {
		return nil, err
	}
	set := make(map[string]bool, len(rows))
	for _, r := range rows {
		set[r.ID] = true
	}
	return set, nil
}

// Upsert inserts/updates rows keyed by primary key id. Returns error message on failure.
func (c *Client) Upsert(ctx context.Context, table string, rows any) error {
	u := c.tableURL(table, "on_conflict=id")
	data, status, err := c.req(ctx, http.MethodPost, u, rows, "resolution=merge-duplicates,return=minimal")
	if err != nil {
		return err
	}
	if status >= 300 {
		return fmt.Errorf("supabase upsert %s: status %d: %s", table, status, truncate(string(data)))
	}
	return nil
}

// DeleteByIDs deletes rows by primary key id. count header dropped.
func (c *Client) DeleteByIDs(ctx context.Context, table string, ids []string) error {
	u := c.tableURL(table, fmt.Sprintf("id=in.(%s)", strings.Join(ids, ",")))
	data, status, err := c.req(ctx, http.MethodDelete, u, nil, "return=minimal")
	if err != nil {
		return err
	}
	if status >= 300 {
		return fmt.Errorf("supabase delete %s: status %d: %s", table, status, truncate(string(data)))
	}
	return nil
}

// Upload uploads file content to the storage bucket and returns its public URL.
func (c *Client) Upload(ctx context.Context, key string, contentType string, content []byte) (string, error) {
	u := fmt.Sprintf("%s/object/%s/%s", c.storage, c.bucket, key)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(content))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.key)
	req.Header.Set("apikey", c.key)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("x-upsert", "true")
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("supabase upload: status %d: %s", resp.StatusCode, truncate(string(body)))
	}
	return fmt.Sprintf("%s/object/public/%s/%s", c.storage, c.bucket, key), nil
}

func truncate(s string) string {
	if len(s) > 500 {
		return s[:500]
	}
	return s
}
