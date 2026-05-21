package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/tejokumar/ai-gateway-mesh/internal/config"
	"github.com/tejokumar/ai-gateway-mesh/internal/openai"
)

type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{httpClient: http.DefaultClient}
}

func NewClientWithHTTPClient(httpClient *http.Client) *Client {
	return &Client{httpClient: httpClient}
}

func (c *Client) Forward(ctx context.Context, backend config.Backend, req openai.ChatCompletionRequest) (*http.Response, error) {
	req.Model = backend.Model
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal backend request: %w", err)
	}

	attempts := backend.MaxRetries + 1
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		resp, err := c.forwardOnce(ctx, backend, body)
		if err != nil {
			lastErr = err
			if attempt < attempts-1 {
				continue
			}
			return nil, err
		}
		if !RetryableStatus(resp.StatusCode) || attempt == attempts-1 {
			return resp, nil
		}
		resp.Body.Close()
	}
	return nil, lastErr
}

func (c *Client) forwardOnce(parent context.Context, backend config.Backend, body []byte) (*http.Response, error) {
	ctx := parent
	cancel := func() {}
	if backend.TimeoutMS > 0 {
		ctx, cancel = context.WithTimeout(parent, time.Duration(backend.TimeoutMS)*time.Millisecond)
	}

	url := strings.TrimRight(backend.BaseURL, "/") + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		cancel()
		return nil, fmt.Errorf("create backend request: %w", err)
	}
	httpReq.Header.Set("content-type", "application/json")
	if backend.APIKeyEnv != "" {
		if apiKey := os.Getenv(backend.APIKeyEnv); apiKey != "" {
			httpReq.Header.Set("authorization", "Bearer "+apiKey)
		}
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("call backend %q: %w", backend.Name, err)
	}
	resp.Body = &cancelOnCloseReadCloser{ReadCloser: resp.Body, cancel: cancel}
	return resp, nil
}

func RetryableStatus(status int) bool {
	return status == http.StatusTooManyRequests ||
		status == http.StatusInternalServerError ||
		status == http.StatusBadGateway ||
		status == http.StatusServiceUnavailable
}

func IsTimeout(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, context.DeadlineExceeded) || strings.Contains(err.Error(), "Client.Timeout")
}

type cancelOnCloseReadCloser struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (r *cancelOnCloseReadCloser) Close() error {
	err := r.ReadCloser.Close()
	r.cancel()
	return err
}
