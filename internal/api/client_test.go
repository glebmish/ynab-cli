package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetRequestBearerAuth(t *testing.T) {
	var capturedReq *http.Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedReq = r
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token", "last-used")
	resp, err := client.Do("GET", "/plans/{plan_id}/accounts", map[string]string{"foo": "bar"}, nil)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer resp.Body.Close()

	if capturedReq.Method != "GET" {
		t.Errorf("Method = %q", capturedReq.Method)
	}
	if !strings.Contains(capturedReq.URL.Path, "last-used") {
		t.Errorf("Path %q should contain last-used", capturedReq.URL.Path)
	}
	if capturedReq.URL.Query().Get("foo") != "bar" {
		t.Errorf("Query foo = %q", capturedReq.URL.Query().Get("foo"))
	}
	auth := capturedReq.Header.Get("Authorization")
	if auth != "Bearer test-token" {
		t.Errorf("Authorization = %q, want Bearer test-token", auth)
	}
}

func TestPostWithJSONBody(t *testing.T) {
	var capturedReq *http.Request
	var capturedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedReq = r
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &capturedBody)
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	client := NewClient(server.URL, "tok", "plan1")
	payload, _ := json.Marshal(map[string]string{"name": "Checking"})
	resp, err := client.Do("POST", "/plans/{plan_id}/accounts", nil, payload)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer resp.Body.Close()

	if capturedReq.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q", capturedReq.Header.Get("Content-Type"))
	}
	if capturedBody["name"] != "Checking" {
		t.Errorf("body name = %v", capturedBody["name"])
	}
}

func TestPathSubstitution(t *testing.T) {
	var capturedPath, capturedQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, "tok", "p1")
	resp, err := client.Do("GET", "/plans/{plan_id}/accounts/{account_id}", map[string]string{"account_id": "acc42"}, nil)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer resp.Body.Close()

	if !strings.Contains(capturedPath, "p1") || !strings.Contains(capturedPath, "acc42") {
		t.Errorf("Path %q missing expected segments", capturedPath)
	}
	if strings.Contains(capturedPath, "{") {
		t.Errorf("Path %q should not contain placeholders", capturedPath)
	}
	if strings.Contains(capturedQuery, "account_id") {
		t.Errorf("Query %q should not include consumed path param", capturedQuery)
	}
}

func TestAPIError401(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"id":"401","name":"unauthorized"}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "bad", "p1")
	_, err := client.Do("GET", "/user", nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("error type = %T", err)
	}
	if apiErr.StatusCode != 401 {
		t.Errorf("StatusCode = %d", apiErr.StatusCode)
	}
	if !strings.Contains(apiErr.Error(), "access token") {
		t.Errorf("Error message missing 401 hint: %s", apiErr.Error())
	}
}

func TestAPIError429Hint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	client := NewClient(server.URL, "tok", "p1")
	_, err := client.Do("GET", "/user", nil, nil)
	apiErr, _ := err.(*APIError)
	if apiErr == nil || !strings.Contains(apiErr.Error(), "rate limited") {
		t.Errorf("expected rate limited hint, got %v", err)
	}
}

func TestDryRun(t *testing.T) {
	client := NewClient("https://api.ynab.com/v1", "tok", "last-used")
	out := client.DryRun("GET", "/plans/{plan_id}/transactions", map[string]string{"since_date": "2025-01-01"}, nil)
	if !strings.Contains(out, "GET") || !strings.Contains(out, "last-used") || !strings.Contains(out, "since_date=2025-01-01") {
		t.Errorf("DryRun output unexpected: %q", out)
	}
}
