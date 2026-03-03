package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// NewAnthropicProxy creates a reverse proxy for the Anthropic API.
// It strips the "/anthropic" prefix from the request path and forwards to baseURL.
// Example: /anthropic/v1/messages → https://api.anthropic.com/v1/messages
func NewAnthropicProxy(baseURL string) (http.Handler, error) {
	target, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	// Capture the default director so we can call it first.
	defaultDirector := proxy.Director

	proxy.Director = func(req *http.Request) {
		// Call the default director to set scheme, host, and standard headers.
		defaultDirector(req)

		// Strip the "/anthropic" prefix so the upstream receives the bare path.
		req.URL.Path = strings.TrimPrefix(req.URL.Path, "/anthropic")
		if req.URL.RawPath != "" {
			req.URL.RawPath = strings.TrimPrefix(req.URL.RawPath, "/anthropic")
		}

		// Override Host so the upstream sees its own hostname, not the proxy's.
		req.Host = target.Host
	}

	return proxy, nil
}
