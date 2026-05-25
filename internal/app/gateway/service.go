package gateway

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"xiaoheiproxy/internal/app/ports"
	"xiaoheiproxy/internal/domain"
	"xiaoheiproxy/internal/infrastructure/upstream"
)

type Service struct {
	mu           sync.RWMutex
	upstream     *upstream.Client
	logs         ports.RequestLogRepository
	cache        ports.Cache
	cacheTTL     time.Duration
	maxBodyBytes int64
	redactor     *Redactor
}

type apiEnvelope struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
	Time int64           `json:"time"`
}

type RuntimeConfig struct {
	UpstreamBaseURL string
	UpstreamAPIKey  string
	Timeout         time.Duration
	CacheTTL        time.Duration
	MaxBodyBytes    int64
	RedactFields    []string
}

type RequestContext struct {
	TraceID     string
	ClientIP    string
	Method      string
	Path        string
	Headers     http.Header
	Body        []byte
	BypassCache bool
}

type Response struct {
	StatusCode  int
	Headers     http.Header
	Body        []byte
	ContentType string
}

func NewService(up *upstream.Client, logs ports.RequestLogRepository, cache ports.Cache, cacheTTL time.Duration, maxBodyBytes int64, redactFields []string) *Service {
	return &Service{
		upstream:     up,
		logs:         logs,
		cache:        cache,
		cacheTTL:     cacheTTL,
		maxBodyBytes: maxBodyBytes,
		redactor:     NewRedactor(redactFields),
	}
}

func (s *Service) Handle(ctx context.Context, req RequestContext) (*Response, error) {
	start := time.Now()
	s.mu.RLock()
	up := s.upstream
	cacheTTL := s.cacheTTL
	redactor := s.redactor
	s.mu.RUnlock()
	logEntry := domain.RequestLog{
		TraceID:           req.TraceID,
		ClientIP:          req.ClientIP,
		DownstreamMethod:  req.Method,
		DownstreamPath:    req.Path,
		DownstreamHeaders: redactor.HeaderJSON(req.Headers),
		DownstreamBody:    redactor.BodyJSON(req.Body),
		CreatedAt:         start,
	}
	defer func() {
		logEntry.DurationMillis = time.Since(start).Milliseconds()
		_ = s.logs.Create(context.Background(), &logEntry)
	}()

	mapped := MapV2ToV1(req.Path, req.Body)
	if !mapped.Supported {
		errMsg := mapped.Reason
		if strings.TrimSpace(errMsg) == "" {
			errMsg = fmt.Sprintf("unsupported v2 endpoint: %s", req.Path)
		}
		err := fmt.Errorf("%s", errMsg)
		resp := errorResponse(http.StatusNotImplemented, err.Error())
		logEntry.ResponseStatus = resp.StatusCode
		logEntry.ResponseBody = string(resp.Body)
		logEntry.Success = false
		logEntry.Message = err.Error()
		return resp, nil
	}
	logEntry.UpstreamMethod = mapped.Request.Method
	cacheKey := s.cacheKey(req.Path, req.Body)
	if mapped.Cacheable && !req.BypassCache {
		if cached, ok := s.cache.Get(cacheKey); ok {
			resp := &Response{StatusCode: http.StatusOK, Headers: jsonHeaders(), Body: cached, ContentType: "application/json"}
			logEntry.ResponseStatus = resp.StatusCode
			logEntry.ResponseHeaders = redactor.HeaderJSON(resp.Headers)
			logEntry.ResponseBody = redactor.BodyJSON(resp.Body)
			logEntry.Success = true
			logEntry.CacheHit = true
			logEntry.Message = "cache hit"
			return resp, nil
		}
	}

	upResp, err := up.Do(ctx, mapped.Request)
	logEntry.UpstreamURL = upstreamURL(upResp)
	logEntry.UpstreamHeaders = redactor.HeaderJSON(upstreamHeaders(upResp))
	logEntry.UpstreamBody = redactor.BodyJSON(mapped.Request.Body)
	if err != nil {
		resp := errorResponse(http.StatusBadGateway, err.Error())
		logEntry.ResponseStatus = resp.StatusCode
		logEntry.ResponseHeaders = redactor.HeaderJSON(resp.Headers)
		logEntry.ResponseBody = redactor.BodyJSON(resp.Body)
		logEntry.Success = false
		logEntry.Message = err.Error()
		return resp, nil
	}
	upstreamSuccess := isUpstreamSuccess(req.Path, upResp.StatusCode, upResp.Body)
	resp := normalizeResponse(req.Path, req.Body, upResp)
	logEntry.UpstreamURL = upResp.URL
	logEntry.UpstreamMethod = upResp.Method
	logEntry.UpstreamHeaders = redactor.HeaderJSON(upResp.RequestHdr)
	logEntry.UpstreamRespStatus = upResp.StatusCode
	logEntry.UpstreamRespHeaders = redactor.HeaderJSON(upResp.Headers)
	logEntry.UpstreamRespBody = redactor.BodyJSON(upResp.Body)
	logEntry.ResponseStatus = resp.StatusCode
	logEntry.ResponseHeaders = redactor.HeaderJSON(resp.Headers)
	logEntry.ResponseBody = redactor.BodyJSON(resp.Body)
	logEntry.Success = upstreamSuccess
	logEntry.Message = extractMessage(resp.Body, logEntry.Success)
	if mapped.Cacheable && logEntry.Success {
		s.cache.Set(cacheKey, resp.Body, cacheTTL)
	}
	if !mapped.Cacheable {
		s.cache.DeletePrefix("gateway:")
	}
	return resp, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func cloneValues(src url.Values) url.Values {
	if src == nil {
		return nil
	}
	dst := make(url.Values, len(src))
	for key, vals := range src {
		dst[key] = append([]string(nil), vals...)
	}
	return dst
}

func (s *Service) UpdateConfig(cfg RuntimeConfig) {
	s.mu.Lock()
	s.upstream = upstream.NewClient(cfg.UpstreamBaseURL, cfg.UpstreamAPIKey, cfg.Timeout)
	s.cacheTTL = cfg.CacheTTL
	s.maxBodyBytes = cfg.MaxBodyBytes
	s.redactor = NewRedactor(cfg.RedactFields)
	s.mu.Unlock()
	s.cache.DeletePrefix("gateway:")
}

func normalizeResponse(path string, requestBody []byte, up *upstream.Response) *Response {
	headers := cloneResponseHeaders(up.Headers)
	body := up.Body
	status := up.StatusCode
	if status >= 300 && status < 400 {
		if loc := up.Headers.Get("Location"); loc != "" {
			body = mustJSON(map[string]any{
				"code": 1,
				"msg":  "ok",
				"time": time.Now().Unix(),
				"data": map[string]any{"url": resolveRelative(up.URL, loc)},
			})
			status = http.StatusOK
			headers = jsonHeaders()
		}
	}
	if len(body) == 0 && strings.HasSuffix(path, "/test") {
		body = mustJSON(map[string]any{"code": 1, "msg": "ok", "time": time.Now().Unix(), "data": map[string]any{}})
		status = http.StatusOK
		headers = jsonHeaders()
	}
	if normalized := normalizeEnvelopeBody(path, requestBody, up.Body, status); normalized != nil {
		body = normalized
		status = http.StatusOK
		headers = jsonHeaders()
	}
	return &Response{StatusCode: status, Headers: headers, Body: body, ContentType: headers.Get("Content-Type")}
}

func errorResponse(status int, msg string) *Response {
	return &Response{
		StatusCode: status,
		Headers:    jsonHeaders(),
		Body: mustJSON(map[string]any{
			"code": 0,
			"msg":  msg,
			"time": time.Now().Unix(),
			"data": map[string]any{},
		}),
		ContentType: "application/json",
	}
}

func (s *Service) cacheKey(path string, body []byte) string {
	sum := sha256.Sum256(bytes.Join([][]byte{[]byte(path), body}, []byte{0}))
	return "gateway:" + hex.EncodeToString(sum[:])
}

func extractMessage(body []byte, success bool) string {
	var payload map[string]any
	if json.Unmarshal(body, &payload) == nil {
		for _, key := range []string{"msg", "message", "error"} {
			if v, ok := payload[key]; ok && strings.TrimSpace(fmt.Sprint(v)) != "" {
				return strings.TrimSpace(fmt.Sprint(v))
			}
		}
	}
	if success {
		return "ok"
	}
	return "upstream error"
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

func jsonHeaders() http.Header {
	h := http.Header{}
	h.Set("Content-Type", "application/json; charset=utf-8")
	return h
}

func cloneResponseHeaders(h http.Header) http.Header {
	out := http.Header{}
	for k, v := range h {
		if strings.EqualFold(k, "Content-Length") || strings.EqualFold(k, "Transfer-Encoding") {
			continue
		}
		out[k] = append([]string(nil), v...)
	}
	if out.Get("Content-Type") == "" {
		out.Set("Content-Type", "application/json; charset=utf-8")
	}
	return out
}

func upstreamURL(resp *upstream.Response) string {
	if resp == nil {
		return ""
	}
	return resp.URL
}

func upstreamHeaders(resp *upstream.Response) http.Header {
	if resp == nil {
		return http.Header{}
	}
	return resp.RequestHdr
}

func resolveRelative(base, loc string) string {
	if strings.HasPrefix(loc, "http://") || strings.HasPrefix(loc, "https://") {
		return loc
	}
	return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(loc, "/")
}

var ErrBodyTooLarge = errors.New("request body too large")

type Redactor struct {
	fields map[string]struct{}
}

func NewRedactor(fields []string) *Redactor {
	m := map[string]struct{}{}
	for _, field := range fields {
		field = strings.ToLower(strings.TrimSpace(field))
		if field != "" {
			m[field] = struct{}{}
		}
	}
	return &Redactor{fields: m}
}

func (r *Redactor) HeaderJSON(h http.Header) string {
	out := map[string]string{}
	keys := make([]string, 0, len(h))
	for key := range h {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if r.isSecret(key) {
			out[key] = "***"
			continue
		}
		out[key] = strings.Join(h.Values(key), ",")
	}
	return string(mustJSON(out))
}

func (r *Redactor) BodyJSON(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	var v any
	if json.Unmarshal(body, &v) == nil {
		return string(mustJSON(r.mask(v)))
	}
	values, err := url.ParseQuery(string(body))
	if err != nil {
		return string(body)
	}
	out := map[string]string{}
	for key, vals := range values {
		if r.isSecret(key) {
			out[key] = "***"
		} else {
			out[key] = strings.Join(vals, ",")
		}
	}
	return string(mustJSON(out))
}

func (r *Redactor) mask(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := map[string]any{}
		for key, val := range t {
			if r.isSecret(key) {
				out[key] = "***"
			} else {
				out[key] = r.mask(val)
			}
		}
		return out
	case []any:
		out := make([]any, 0, len(t))
		for _, item := range t {
			out = append(out, r.mask(item))
		}
		return out
	default:
		return t
	}
}

func (r *Redactor) isSecret(key string) bool {
	k := strings.ToLower(strings.TrimSpace(key))
	if _, ok := r.fields[k]; ok {
		return true
	}
	return strings.Contains(k, "password") || strings.Contains(k, "secret") || strings.Contains(k, "token")
}
