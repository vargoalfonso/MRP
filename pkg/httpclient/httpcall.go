// Package httpclient provides a minimal HTTP client used for outbound API calls.
package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"time"

	"github.com/ganasa18/go-template/pkg/logger"
)

// ClientFactory creates Client instances.
type ClientFactory interface {
	CreateClient() Client
}

// Client abstracts outbound HTTP requests.
type Client interface {
	GetJSON(ctx context.Context, url string, headers map[string]string, dest interface{}) (int, error)
	PostJSON(ctx context.Context, url, bodyJSON string, headers map[string]string, dest interface{}) (int, error)
}

type clientFactory struct{}

// New returns a ClientFactory.
func New() ClientFactory { return &clientFactory{} }

func (c clientFactory) CreateClient() Client {
	return &client{timeout: 30 * time.Second}
}

type client struct {
	timeout time.Duration
}

func (c *client) GetJSON(ctx context.Context, url string, headers map[string]string, dest interface{}) (int, error) {
	log := logger.FromContext(ctx)
	log.Info("outbound GET", slog.String("url", url))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return c.do(ctx, req, dest)
}

func (c *client) PostJSON(ctx context.Context, url, bodyJSON string, headers map[string]string, dest interface{}) (int, error) {
	log := logger.FromContext(ctx)
	log.Info("outbound POST", slog.String("url", url))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBufferString(bodyJSON))
	if err != nil {
		return http.StatusInternalServerError, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return c.do(ctx, req, dest)
}

func (c *client) do(ctx context.Context, req *http.Request, dest interface{}) (int, error) {
	hc := &http.Client{Timeout: c.timeout}
	resp, err := hc.Do(req)
	if err != nil {
		if resp == nil {
			return 0, err
		}
		return resp.StatusCode, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if len(body) == 0 {
		return resp.StatusCode, errors.New(resp.Status)
	}

	if err = json.Unmarshal(body, dest); err != nil {
		nl := regexp.MustCompile(`[\r\n]+`)
		return resp.StatusCode, fmt.Errorf("response decode failed, body: %s", nl.ReplaceAllString(string(body), " "))
	}

	logger.FromContext(ctx).Info("outbound response",
		slog.Int("status", resp.StatusCode),
		slog.String("url", req.URL.String()),
	)
	return resp.StatusCode, nil
}
