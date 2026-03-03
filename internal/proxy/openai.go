package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// NewOpenAIProxy creates a reverse proxy for the OpenAI API.
// It strips the "/openai" prefix from the request path and forwards to baseURL.
// Example: /openai/v1/chat/completions → https://api.openai.com/v1/chat/completions
func NewOpenAIProxy(baseURL string) (http.Handler, error) {
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

		// Strip the "/openai" prefix so the upstream receives the bare path.
		req.URL.Path = strings.TrimPrefix(req.URL.Path, "/openai")
		if req.URL.RawPath != "" {
			req.URL.RawPath = strings.TrimPrefix(req.URL.RawPath, "/openai")
		}

		// Override Host so the upstream sees its own hostname, not the proxy's.
		req.Host = target.Host
	}

	return proxy, nil
}
