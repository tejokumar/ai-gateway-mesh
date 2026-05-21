package gatewayhttp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/tejokumar/ai-gateway-mesh/internal/backend"
	"github.com/tejokumar/ai-gateway-mesh/internal/config"
	"github.com/tejokumar/ai-gateway-mesh/internal/metrics"
	"github.com/tejokumar/ai-gateway-mesh/internal/openai"
	"github.com/tejokumar/ai-gateway-mesh/internal/router"
)

type Server struct {
	cfg           config.Config
	router        *router.Router
	backendClient *backend.Client
	metrics       *metrics.Metrics
	logger        *slog.Logger
	mux           *http.ServeMux
}

func New(cfg config.Config, gatewayRouter *router.Router, backendClient *backend.Client, m *metrics.Metrics, logger *slog.Logger) *Server {
	s := &Server{
		cfg:           cfg,
		router:        gatewayRouter,
		backendClient: backendClient,
		metrics:       m,
		logger:        logger,
		mux:           http.NewServeMux(),
	}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler {
	return s.withMetrics(s.mux)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /", s.index)
	s.mux.HandleFunc("GET /healthz", s.healthz)
	s.mux.HandleFunc("GET /readyz", s.readyz)
	s.mux.Handle("GET /metrics", s.metrics.Handler())
	s.mux.HandleFunc("GET /admin/backends", s.adminBackends)
	s.mux.HandleFunc("GET /admin/policies", s.adminPolicies)
	s.mux.HandleFunc("POST /v1/chat/completions", s.chatCompletions)
}

func (s *Server) index(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"name":   "AI Gateway Mesh",
		"status": "ok",
		"endpoints": []string{
			"GET /healthz",
			"GET /readyz",
			"GET /metrics",
			"GET /admin/backends",
			"GET /admin/policies",
			"POST /v1/chat/completions",
		},
	})
}

func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) readyz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *Server) adminBackends(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.cfg.SanitizedBackends())
}

func (s *Server) adminPolicies(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.cfg.Policies)
}

func (s *Server) chatCompletions(w http.ResponseWriter, r *http.Request) {
	s.metrics.InflightRequests.Inc()
	defer s.metrics.InflightRequests.Dec()

	started := time.Now()
	var status = http.StatusOK
	var selectedBackend string
	defer func() {
		s.metrics.Observe("/v1/chat/completions", selectedBackend, status, started)
	}()

	var req openai.ChatCompletionRequest
	decoder := json.NewDecoder(io.LimitReader(r.Body, 4<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		status = http.StatusBadRequest
		writeOpenAIError(w, status, "invalid JSON request")
		return
	}
	if err := req.Validate(); err != nil {
		status = http.StatusBadRequest
		writeOpenAIError(w, status, err.Error())
		return
	}

	tokenEstimate := router.EstimatePromptTokens(req.Messages)
	s.metrics.EstimatedPromptTokens.Observe(float64(tokenEstimate))
	route, ok := s.router.Route(router.Input{
		RequestedModel:        req.Model,
		EstimatedPromptTokens: tokenEstimate,
		Messages:              req.Messages,
		Tenant:                r.Header.Get("x-tenant-id"),
	})
	if !ok {
		status = http.StatusServiceUnavailable
		writeOpenAIError(w, status, "no backend available")
		return
	}
	selectedBackend = route.Backend.Name

	resp, responseBackend, usedFallback, err := s.callWithFallback(r.Context(), route.Backend, req)
	if err != nil {
		status = http.StatusBadGateway
		s.logger.Warn("chat completion failed", "backend", selectedBackend, "error", err)
		writeOpenAIError(w, status, "backend request failed")
		return
	}
	defer resp.Body.Close()

	selectedBackend = responseBackend
	status = resp.StatusCode
	w.Header().Set("x-ai-gateway-backend", selectedBackend)
	w.Header().Set("x-ai-gateway-fallback-used", fmt.Sprintf("%t", usedFallback))
	copyResponse(w, resp)
}

func (s *Server) callWithFallback(ctx context.Context, primary config.Backend, req openai.ChatCompletionRequest) (*http.Response, string, bool, error) {
	resp, err := s.backendClient.Forward(ctx, primary, req)
	if err == nil && !backend.RetryableStatus(resp.StatusCode) {
		return resp, primary.Name, false, nil
	}

	if err == nil {
		s.metrics.BackendErrorsTotal.WithLabelValues(primary.Name, fmt.Sprintf("%d", resp.StatusCode)).Inc()
		resp.Body.Close()
	} else if !backend.IsTimeout(err) && primary.Fallback == "" {
		return nil, primary.Name, false, err
	}

	if primary.Fallback == "" {
		if err != nil {
			return nil, primary.Name, false, err
		}
		return nil, primary.Name, false, errors.New("backend returned retryable status without fallback")
	}

	fallback, ok := s.cfg.BackendByName(primary.Fallback)
	if !ok {
		return nil, primary.Name, false, fmt.Errorf("fallback backend %q is not configured", primary.Fallback)
	}
	s.metrics.FallbackTotal.WithLabelValues(primary.Name, fallback.Name).Inc()
	resp, err = s.backendClient.Forward(ctx, fallback, req)
	if err != nil {
		return nil, fallback.Name, true, err
	}
	return resp, fallback.Name, true, nil
}

func (s *Server) withMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/chat/completions" {
			next.ServeHTTP(w, r)
			return
		}
		started := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		s.metrics.InflightRequests.Inc()
		defer s.metrics.InflightRequests.Dec()
		next.ServeHTTP(recorder, r)
		s.metrics.Observe(r.URL.Path, "", recorder.status, started)
	})
}

func copyResponse(w http.ResponseWriter, resp *http.Response) {
	for name, values := range resp.Header {
		if strings.EqualFold(name, "content-length") {
			continue
		}
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}
	if resp.Header.Get("content-type") == "" {
		w.Header().Set("content-type", "application/json")
	}
	w.WriteHeader(resp.StatusCode)

	flusher, canFlush := w.(http.Flusher)
	buf := make([]byte, 32*1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			_, _ = w.Write(buf[:n])
			if canFlush {
				flusher.Flush()
			}
		}
		if readErr != nil {
			break
		}
	}
}

func writeOpenAIError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, openai.ErrorResponse{
		Error: openai.ErrorBody{Message: message, Type: "invalid_request_error"},
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}
