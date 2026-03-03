package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestNewOpenAIProxy_StripPrefix verifies that the "/openai" prefix is stripped
// before the request reaches the upstream server.
func TestNewOpenAIProxy_StripPrefix(t *testing.T) {
	var receivedPath string

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	handler, err := NewOpenAIProxy(upstream.URL)
	if err != nil {
		t.Fatalf("NewOpenAIProxy returned unexpected error: %v", err)
	}

	proxyServer := httptest.NewServer(handler)
	defer proxyServer.Close()

	resp, err := http.Get(proxyServer.URL + "/openai/v1/chat/completions")
	if err != nil {
		t.Fatalf("GET request failed: %v", err)
	}
	resp.Body.Close()

	want := "/v1/chat/completions"
	if receivedPath != want {
		t.Errorf("upstream received path %q; want %q", receivedPath, want)
	}
}

// TestNewAnthropicProxy_StripPrefix verifies that the "/anthropic" prefix is
// stripped before the request reaches the upstream server.
func TestNewAnthropicProxy_StripPrefix(t *testing.T) {
	var receivedPath string

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	handler, err := NewAnthropicProxy(upstream.URL)
	if err != nil {
		t.Fatalf("NewAnthropicProxy returned unexpected error: %v", err)
	}

	proxyServer := httptest.NewServer(handler)
	defer proxyServer.Close()

	resp, err := http.Get(proxyServer.URL + "/anthropic/v1/messages")
	if err != nil {
		t.Fatalf("GET request failed: %v", err)
	}
	resp.Body.Close()

	want := "/v1/messages"
	if receivedPath != want {
		t.Errorf("upstream received path %q; want %q", receivedPath, want)
	}
}

// TestNewOpenAIProxy_HeadersForwarded verifies that the Authorization header
// sent by the client is forwarded unchanged to the upstream server.
func TestNewOpenAIProxy_HeadersForwarded(t *testing.T) {
	var receivedAuth string

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	handler, err := NewOpenAIProxy(upstream.URL)
	if err != nil {
		t.Fatalf("NewOpenAIProxy returned unexpected error: %v", err)
	}

	proxyServer := httptest.NewServer(handler)
	defer proxyServer.Close()

	req, err := http.NewRequest(http.MethodPost, proxyServer.URL+"/openai/v1/chat/completions", nil)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	wantAuth := "Bearer sk-test1234"
	req.Header.Set("Authorization", wantAuth)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()

	if receivedAuth != wantAuth {
		t.Errorf("upstream received Authorization %q; want %q", receivedAuth, wantAuth)
	}
}

// TestNewAnthropicProxy_HeadersForwarded verifies that the x-api-key header
// sent by the client is forwarded unchanged to the upstream server.
func TestNewAnthropicProxy_HeadersForwarded(t *testing.T) {
	var receivedAPIKey string

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAPIKey = r.Header.Get("x-api-key")
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	handler, err := NewAnthropicProxy(upstream.URL)
	if err != nil {
		t.Fatalf("NewAnthropicProxy returned unexpected error: %v", err)
	}

	proxyServer := httptest.NewServer(handler)
	defer proxyServer.Close()

	req, err := http.NewRequest(http.MethodPost, proxyServer.URL+"/anthropic/v1/messages", nil)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	wantKey := "sk-ant-test5678"
	req.Header.Set("x-api-key", wantKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()

	if receivedAPIKey != wantKey {
		t.Errorf("upstream received x-api-key %q; want %q", receivedAPIKey, wantKey)
	}
}
