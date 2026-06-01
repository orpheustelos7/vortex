package gateway

import (
"context"
"fmt"
"log"
"net/http"
"net/http/httputil"
"net/url"
"strconv"
"time"

"github.com/orpheustelos7/vortex/internal/auth"
"github.com/orpheustelos7/vortex/internal/config"
"github.com/orpheustelos7/vortex/internal/observability"
"github.com/orpheustelos7/vortex/internal/ratelimiter"
)

type Server struct {
store   *config.Store
limiter ratelimiter.Limiter
metrics *observability.Metrics
}

func New(store *config.Store, limiter ratelimiter.Limiter, metrics *observability.Metrics) *Server {
return &Server{store: store, limiter: limiter, metrics: metrics}
}

func (s *Server) Handler() http.Handler {
mux := http.NewServeMux()
mux.Handle("/metrics", s.metrics.Handler())
mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
w.WriteHeader(http.StatusOK)
_, _ = w.Write([]byte("ok"))
})
mux.HandleFunc("/", s.handleProxy)
return mux
}

func (s *Server) handleProxy(w http.ResponseWriter, r *http.Request) {
started := time.Now()
tenant := r.Header.Get("X-Tenant-ID")
if tenant == "" {
http.Error(w, "missing X-Tenant-ID", http.StatusBadRequest)
s.metrics.Errors.WithLabelValues("missing_tenant").Inc()
s.metrics.Observe("unknown", http.StatusBadRequest, started)
return
}

policy, ok := s.store.Get(tenant)
if !ok {
http.Error(w, "tenant policy not found", http.StatusNotFound)
s.metrics.Errors.WithLabelValues("missing_policy").Inc()
s.metrics.Observe(tenant, http.StatusNotFound, started)
return
}

if err := auth.ValidateRequest(r, policy); err != nil {
http.Error(w, "unauthorized", http.StatusUnauthorized)
s.metrics.Errors.WithLabelValues("auth").Inc()
s.metrics.Observe(tenant, http.StatusUnauthorized, started)
return
}

result, err := s.limiter.Allow(r.Context(), tenant, policy.Rate, policy.Burst)
if err != nil {
http.Error(w, "rate limiter unavailable", http.StatusServiceUnavailable)
s.metrics.Errors.WithLabelValues("ratelimiter").Inc()
s.metrics.Observe(tenant, http.StatusServiceUnavailable, started)
return
}
if !result.Allowed {
w.Header().Set("Retry-After", strconv.Itoa(max(1, int(result.RetryAfter.Seconds()))))
http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
s.metrics.RateLimited.Inc()
s.metrics.Observe(tenant, http.StatusTooManyRequests, started)
return
}
w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(result.Remaining, 10))

target, err := url.Parse(policy.BackendURL)
if err != nil {
http.Error(w, "invalid backend", http.StatusInternalServerError)
s.metrics.Errors.WithLabelValues("backend_url").Inc()
s.metrics.Observe(tenant, http.StatusInternalServerError, started)
return
}

proxy := httputil.NewSingleHostReverseProxy(target)
proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
log.Printf("proxy error tenant=%s: %v", tenant, err)
http.Error(w, "upstream error", http.StatusBadGateway)
s.metrics.Errors.WithLabelValues("upstream").Inc()
s.metrics.Observe(tenant, http.StatusBadGateway, started)
}
proxy.ModifyResponse = func(resp *http.Response) error {
s.metrics.Observe(tenant, resp.StatusCode, started)
return nil
}

ctx := context.WithValue(r.Context(), struct{}{}, fmt.Sprintf("tenant:%s", tenant))
proxy.ServeHTTP(w, r.WithContext(ctx))
}
