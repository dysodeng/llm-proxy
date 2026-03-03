# LLM Proxy — Design Document

**Date:** 2026-03-03
**Status:** Approved

---

## Overview

`llm-proxy` is a Go-based HTTP reverse proxy for OpenAI and Anthropic APIs. It provides transparent request forwarding, per-API-key rate limiting, structured request logging, and a web dashboard.

**Non-goals:** Protocol conversion between OpenAI and Anthropic formats is explicitly out of scope.

---

## Architecture

Selected approach: **`httputil.ReverseProxy`** (standard library reverse proxy).

- Natively supports SSE (Server-Sent Events) streaming responses — critical for LLM chat completions
- Minimal external dependencies
- Standard Go middleware pattern (`http.Handler` wrapping)

---

## URL Routing

| Path | Behavior |
|------|----------|
| `GET /` | Web dashboard: usage instructions + runtime statistics |
| `/openai/*` | Reverse proxy to `https://api.openai.com` (full path preserved) |
| `/anthropic/*` | Reverse proxy to `https://api.anthropic.com` (full path preserved) |

**Example:**
- `POST /openai/v1/chat/completions` → `https://api.openai.com/v1/chat/completions`
- `POST /anthropic/v1/messages` → `https://api.anthropic.com/v1/messages`

All request headers (including `Authorization`, `x-api-key`) are forwarded unchanged. No header modification.

---

## Request Processing Pipeline

```
HTTP Request
    ↓
[Logging Middleware]     ← Record request start; on completion log latency/status
    ↓
[Rate Limit Middleware]  ← Extract API key; check/update token bucket
    ↓
[ReverseProxy Handler]  ← Forward request; streaming passthrough
    ↓
Upstream (OpenAI / Anthropic)
```

---

## Rate Limiting

- **Algorithm:** Token bucket (`golang.org/x/time/rate`)
- **Granularity:** Per API key
- **Key extraction:**
  - OpenAI: `Authorization: Bearer <key>`
  - Anthropic: `x-api-key: <key>` or `Authorization: Bearer <key>`
- **Storage:** `sync.Map` mapping key → `rate.Limiter`
- **Whitelist:** Configured keys bypass rate limiting entirely
- **Overrides:** Per-key custom rate limit configuration
- **429 response** returned when limit is exceeded

---

## Logging

- **Library:** `go.uber.org/zap` + `gopkg.in/natefinish/lumberjack.v2`
- **Output:** stdout + rotating file
- **Rotation:** Daily (by date), configurable retention period

**Per-request log fields:**
```
timestamp | provider | path | api_key(last 4 chars masked) | status_code | latency_ms | req_bytes | resp_bytes
```

---

## Configuration (`config.yaml`)

```yaml
server:
  port: 8080

log:
  level: info          # debug | info | warn | error
  file: ./logs/proxy.log
  max_age: 30          # days to retain log files

rate_limit:
  enabled: true
  default:
    requests_per_second: 10
    burst: 20
  whitelist:           # keys exempt from rate limiting
    - "sk-xxxx"
  overrides:           # per-key custom limits
    "sk-zzzz":
      requests_per_second: 100
      burst: 200

providers:
  openai:
    base_url: "https://api.openai.com"
  anthropic:
    base_url: "https://api.anthropic.com"
```

---

## Dashboard (`GET /`)

Static HTML page embedded in binary via `go:embed`. Sections:

1. **Usage Guide** — curl examples for OpenAI and Anthropic endpoints
2. **Runtime Status** — uptime, version
3. **Request Statistics** — total requests, per-provider counts (in-memory atomic counters, reset on restart)
4. **Rate Limit Overview** — current default limits (no key values exposed)

---

## Project Structure

```
llm-proxy/
├── cmd/
│   └── proxy/
│       └── main.go              # Entry point
├── internal/
│   ├── config/
│   │   └── config.go            # Config loading (viper)
│   ├── logger/
│   │   └── logger.go            # zap + lumberjack setup
│   ├── middleware/
│   │   ├── logging.go           # Request logging middleware
│   │   └── ratelimit.go         # Rate limiting middleware
│   ├── proxy/
│   │   ├── openai.go            # OpenAI ReverseProxy builder
│   │   └── anthropic.go         # Anthropic ReverseProxy builder
│   ├── server/
│   │   └── server.go            # HTTP server assembly and routing
│   └── dashboard/
│       └── handler.go           # Dashboard page handler
├── web/
│   └── index.html               # Dashboard HTML (embedded)
├── config.yaml
├── go.mod
└── go.sum
```

---

## Dependencies

| Package | Purpose |
|---------|---------|
| `go.uber.org/zap` | Structured logging |
| `gopkg.in/natefinish/lumberjack.v2` | Log file rotation |
| `golang.org/x/time/rate` | Token bucket rate limiting |
| `github.com/spf13/viper` | Configuration loading |

---

## Error Handling

- Upstream errors (5xx, timeouts): proxied as-is to downstream
- Rate limit exceeded: `429 Too Many Requests` with JSON body
- Invalid/missing provider path: `404 Not Found`
- Config load failure: fatal error at startup

---

## Testing Strategy

- Unit tests for rate limiter logic (whitelist, overrides, key extraction)
- Unit tests for config loading
- Integration tests using `httptest` to verify proxy forwarding headers
- Manual verification of SSE streaming with a real upstream call
