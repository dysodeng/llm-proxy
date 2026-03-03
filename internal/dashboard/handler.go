package dashboard

import (
	_ "embed"
	"encoding/json"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"llm-proxy/internal/config"
)

//go:embed web/index.html
var indexHTML string

// Stats tracks request counts atomically.
type Stats struct {
	Total     atomic.Int64
	OpenAI    atomic.Int64
	Anthropic atomic.Int64
}

// Handler serves the dashboard HTML page.
type Handler struct {
	stats     *Stats
	startTime time.Time
	cfg       config.RateLimitConfig
	version   string
	baseURL   string
}

// NewHandler creates a new dashboard Handler.
func NewHandler(stats *Stats, cfg config.RateLimitConfig, version, baseURL string) *Handler {
	return &Handler{
		stats:     stats,
		startTime: time.Now(),
		cfg:       cfg,
		version:   version,
		baseURL:   baseURL,
	}
}

// proxyData is the JSON payload injected into the HTML page.
type proxyData struct {
	Uptime    string        `json:"uptime"`
	Version   string        `json:"version"`
	BaseURL   string        `json:"base_url"`
	Stats     statsData     `json:"stats"`
	RateLimit rateLimitData `json:"rate_limit"`
}

type statsData struct {
	Total     int64 `json:"total"`
	OpenAI    int64 `json:"openai"`
	Anthropic int64 `json:"anthropic"`
}

type rateLimitData struct {
	RequestsPerSecond float64 `json:"requests_per_second"`
	Burst             int     `json:"burst"`
}

// ServeHTTP serves the dashboard page.
// It injects a <script> tag with window.__PROXY_DATA__ JSON before </head>.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(h.startTime).Round(time.Second).String()

	data := proxyData{
		Uptime:  uptime,
		Version: h.version,
		BaseURL: h.baseURL,
		Stats: statsData{
			Total:     h.stats.Total.Load(),
			OpenAI:    h.stats.OpenAI.Load(),
			Anthropic: h.stats.Anthropic.Load(),
		},
		RateLimit: rateLimitData{
			RequestsPerSecond: h.cfg.Default.RequestsPerSecond,
			Burst:             h.cfg.Default.Burst,
		},
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	scriptTag := "<script>\nwindow.__PROXY_DATA__ = " + string(jsonBytes) + ";\n</script>\n</head>"
	page := strings.Replace(indexHTML, "</head>", scriptTag, 1)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(page))
}
