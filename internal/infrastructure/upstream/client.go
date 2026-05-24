package upstream

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

type Request struct {
	Method      string
	Path        string
	Query       url.Values
	Body        []byte
	ContentType string
}

type Response struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
	URL        string
	Method     string
	RequestHdr http.Header
}

func NewClient(baseURL, apiKey string, timeout time.Duration) *Client {
	return &Client{
		baseURL: normalizeBaseURL(baseURL),
		apiKey:  strings.TrimSpace(apiKey),
		http: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

func (c *Client) Do(ctx context.Context, in Request) (*Response, error) {
	method := strings.ToUpper(strings.TrimSpace(in.Method))
	if method == "" {
		method = http.MethodPost
	}
	fullURL := c.baseURL + "/" + strings.TrimLeft(in.Path, "/")
	if len(in.Query) > 0 {
		sep := "?"
		if strings.Contains(fullURL, "?") {
			sep = "&"
		}
		fullURL += sep + in.Query.Encode()
	}
	var body io.Reader
	if len(in.Body) > 0 {
		body = bytes.NewReader(in.Body)
	}
	req, err := http.NewRequestWithContext(ctx, method, fullURL, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("apikey", c.apiKey)
	if strings.TrimSpace(in.ContentType) != "" && len(in.Body) > 0 {
		req.Header.Set("Content-Type", in.ContentType)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return &Response{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header.Clone(),
		Body:       b,
		URL:        fullURL,
		Method:     method,
		RequestHdr: req.Header.Clone(),
	}, nil
}

func normalizeBaseURL(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return trimmed
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return strings.TrimRight(trimmed, "/")
	}
	if parsed.Path == "" || parsed.Path == "/" {
		parsed.Path = "/index.php/api/cloud"
	}
	return strings.TrimRight(parsed.String(), "/")
}
