package api

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type contextKey struct{}

type APIError struct {
	StatusCode int
	Body       string
	Method     string
	Path       string
}

func (e *APIError) IsAuth() bool {
	return e.StatusCode == 401 || e.StatusCode == 403
}

func (e *APIError) Error() string {
	statusText := http.StatusText(e.StatusCode)
	msg := fmt.Sprintf("API error %d %s: %s %s", e.StatusCode, statusText, e.Method, e.Path)
	if e.Body != "" {
		msg += fmt.Sprintf(": %s", e.Body)
	}
	switch e.StatusCode {
	case 401:
		msg += " (hint: check your access token; YNAB tokens are created at https://app.ynab.com/settings/developer)"
	case 404:
		msg += " (hint: resource not found - verify plan_id/account_id/etc. Use 'ynab plans list' to see valid IDs)"
	case 409:
		msg += " (hint: conflict - resource may already exist or was modified concurrently)"
	case 429:
		msg += " (hint: rate limited - YNAB allows 200 requests/hour per token)"
	}
	return msg
}

type Client struct {
	baseURL     string
	accessToken string
	planID      string
	httpClient  *http.Client
}

func NewClient(baseURL, accessToken, planID string) *Client {
	return &Client{
		baseURL:     baseURL,
		accessToken: accessToken,
		planID:      planID,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
	}
}

func WithContext(ctx context.Context, client *Client) context.Context {
	return context.WithValue(ctx, contextKey{}, client)
}

func FromContext(ctx context.Context) *Client {
	c, _ := ctx.Value(contextKey{}).(*Client)
	return c
}

func (c *Client) Do(method, pathTemplate string, params map[string]string, body []byte) (*http.Response, error) {
	return c.DoWithContext(context.Background(), method, pathTemplate, params, body)
}

func (c *Client) DoWithContext(ctx context.Context, method, pathTemplate string, params map[string]string, body []byte) (*http.Response, error) {
	remaining := make(map[string]string)
	for k, v := range params {
		remaining[k] = v
	}

	path := pathTemplate
	path = strings.ReplaceAll(path, "{plan_id}", c.planID)

	for k, v := range remaining {
		placeholder := "{" + k + "}"
		if strings.Contains(path, placeholder) {
			path = strings.ReplaceAll(path, placeholder, v)
			delete(remaining, k)
		}
	}

	rawURL := c.baseURL + path
	if len(remaining) > 0 {
		q := url.Values{}
		for k, v := range remaining {
			q.Set(k, v)
		}
		rawURL += "?" + q.Encode()
	}

	var bodyReader io.Reader
	if len(body) > 0 {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, rawURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Accept", "application/json")

	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		respBody, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Body:       strings.TrimSpace(string(respBody)),
			Method:     method,
			Path:       path,
		}
	}

	return resp, nil
}

func (c *Client) DryRun(method, pathTemplate string, params map[string]string, body []byte) string {
	remaining := make(map[string]string)
	for k, v := range params {
		remaining[k] = v
	}

	path := pathTemplate
	path = strings.ReplaceAll(path, "{plan_id}", c.planID)

	for k, v := range remaining {
		placeholder := "{" + k + "}"
		if strings.Contains(path, placeholder) {
			path = strings.ReplaceAll(path, placeholder, v)
			delete(remaining, k)
		}
	}

	rawURL := c.baseURL + path
	if len(remaining) > 0 {
		q := url.Values{}
		for k, v := range remaining {
			q.Set(k, v)
		}
		rawURL += "?" + q.Encode()
	}

	result := fmt.Sprintf("%s %s", method, rawURL)
	if len(body) > 0 {
		preview := string(body)
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		result += fmt.Sprintf(" body: %s", preview)
	}
	return result
}
