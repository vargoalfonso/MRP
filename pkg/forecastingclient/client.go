package forecastingclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/ganasa18/go-template/pkg/apperror"
)

const (
	defaultTimeout = 60 * time.Second
	longTimeout    = 120 * time.Second
)

// Options configures the Forecasting client.
type Options struct {
	BaseURL       string        // e.g. https://dev-mrp-forecasting-482804304.asia-southeast1.run.app
	BasicAuthUser string        // if set, Basic Auth header is sent
	BasicAuthPass string
	Timeout       time.Duration // per-request timeout; defaults to 60s
	HTTPClient    *http.Client // optional; defaults to http.DefaultClient
}

// Client wraps HTTP calls to the external forecasting pipeline API.
type Client struct {
	baseURL    string
	basicUser  string
	basicPass  string
	httpClient *http.Client
}

// New builds a Forecasting client from Options.
func New(opts Options) *Client {
	if opts.Timeout == 0 {
		opts.Timeout = defaultTimeout
	}
	hc := opts.HTTPClient
	if hc == nil {
		hc = &http.Client{
			Timeout: opts.Timeout,
		}
	}
	return &Client{
		baseURL:    opts.BaseURL,
		basicUser:  opts.BasicAuthUser,
		basicPass:  opts.BasicAuthPass,
		httpClient: hc,
	}
}

// do is the low-level HTTP helper.
// method: GET/POST/etc, path: relative to baseURL, body: nil for GET, bodyBytes for POST with JSON,
// contentType: "application/json" or "multipart/form-data",
// dest: pointer to struct to decode JSON response.
func (c *Client) do(ctx context.Context, method, path string, bodyBytes []byte, contentType string, dest interface{}) error {
	url := c.baseURL + path

	var body io.Reader
	if len(bodyBytes) > 0 {
		body = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return apperror.InternalWrap("build request failed", err)
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	if c.basicUser != "" {
		req.SetBasicAuth(c.basicUser, c.basicPass)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return apperror.GatewayTimeout("forecasting API unreachable", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MB limit
	if err != nil {
		return apperror.InternalWrap("read response failed", err)
	}

	if resp.StatusCode >= 400 {
		var apiErr APIErrorResponse
		if unmarshalErr := json.Unmarshal(respBody, &apiErr); unmarshalErr == nil && apiErr.Message != "" {
			return apperror.New(resp.StatusCode, apperror.CodeInternalError, apiErr.Message)
		}
		// Fallback: include raw body in message
		return apperror.New(resp.StatusCode, apperror.CodeInternalError, fmt.Sprintf("forecasting API error %d: %s", resp.StatusCode, string(respBody)))
	}

	if dest == nil {
		return nil
	}
	if err := json.Unmarshal(respBody, dest); err != nil {
		return apperror.InternalWrap("decode response failed", err)
	}
	return nil
}

// doLong is like do but uses the long timeout for training triggers etc.
func (c *Client) doLong(ctx context.Context, method, path string, bodyBytes []byte, contentType string, dest interface{}) error {
	hc := &http.Client{Timeout: longTimeout}
	cli := &Client{
		baseURL:    c.baseURL,
		basicUser:  c.basicUser,
		basicPass:  c.basicPass,
		httpClient: hc,
	}
	return cli.do(ctx, method, path, bodyBytes, contentType, dest)
}

// uploadMultipart sends a multipart/form-data POST.
// It appends fields (key, value) and one file field (key, filename, fileBytes).
func (c *Client) uploadMultipart(ctx context.Context, path string, fields map[string]string, fileField, fileName string, fileBytes []byte, dest interface{}) error {
	pr, pw := io.Pipe()
	mpw := multipart.NewWriter(pw)

	go func() {
		defer pw.Close()
		for k, v := range fields {
			if err := mpw.WriteField(k, v); err != nil {
				pw.CloseWithError(err)
				return
			}
		}
		if fileName != "" {
			fw, err := mpw.CreateFormFile(fileField, fileName)
			if err != nil {
				pw.CloseWithError(err)
				return
			}
			if _, err := fw.Write(fileBytes); err != nil {
				pw.CloseWithError(err)
				return
			}
		}
		mpw.Close()
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, pr)
	if err != nil {
		return apperror.InternalWrap("build multipart request failed", err)
	}
	req.Header.Set("Content-Type", mpw.FormDataContentType())
	if c.basicUser != "" {
		req.SetBasicAuth(c.basicUser, c.basicPass)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return apperror.GatewayTimeout("forecasting API unreachable", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return apperror.InternalWrap("read multipart response failed", err)
	}

	if resp.StatusCode >= 400 {
		var apiErr APIErrorResponse
		if unmarshalErr := json.Unmarshal(respBody, &apiErr); unmarshalErr == nil && apiErr.Message != "" {
			return apperror.New(resp.StatusCode, apperror.CodeInternalError, apiErr.Message)
		}
		return apperror.New(resp.StatusCode, apperror.CodeInternalError, fmt.Sprintf("forecasting API error %d: %s", resp.StatusCode, string(respBody)))
	}

	if dest != nil {
		if err := json.Unmarshal(respBody, dest); err != nil {
			return apperror.InternalWrap("decode multipart response failed", err)
		}
	}
	return nil
}

// APIErrorResponse is used to extract error messages from external API error responses.
type APIErrorResponse struct {
	Message string `json:"message"`
	Error   string `json:"error"`
}
