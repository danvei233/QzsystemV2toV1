package httpadapter

import (
	"io"
	"net/http"
	"strings"

	"xiaoheiproxy/internal/app/gateway"
	"xiaoheiproxy/internal/infrastructure/config"

	"github.com/gin-gonic/gin"
)

type GatewayHandler struct {
	service *gateway.Service
	cfg     *config.GatewayConfig
}

func NewGatewayHandler(service *gateway.Service, cfg *config.GatewayConfig) *GatewayHandler {
	return &GatewayHandler{service: service, cfg: cfg}
}

func (h *GatewayHandler) RequireClientKey() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !h.cfg.RequireAPIKey {
			c.Next()
			return
		}
		key := strings.TrimSpace(c.GetHeader("signature"))
		if key == "" {
			key = strings.TrimSpace(c.GetHeader("apikey"))
		}
		if key == "" {
			key = strings.TrimPrefix(strings.TrimSpace(c.GetHeader("Authorization")), "Bearer ")
		}
		if !containsKey(h.cfg.AcceptedAPIKeys, key) {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 0, "msg": "invalid gateway api key"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func containsKey(keys []string, key string) bool {
	for _, item := range keys {
		if strings.TrimSpace(item) == key {
			return true
		}
	}
	return false
}

func (h *GatewayHandler) Handle(c *gin.Context) {
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, h.cfg.MaxBodyBytes+1))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 0, "msg": "read request body failed"})
		return
	}
	if int64(len(body)) > h.cfg.MaxBodyBytes {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"code": 0, "msg": gateway.ErrBodyTooLarge.Error()})
		return
	}
	reqPath := "/api/v1/" + strings.TrimPrefix(c.Param("action"), "/")
	resp, err := h.service.Handle(c.Request.Context(), gateway.RequestContext{
		TraceID:     traceID(c),
		ClientIP:    c.ClientIP(),
		Method:      c.Request.Method,
		Path:        reqPath,
		Headers:     c.Request.Header.Clone(),
		Body:        body,
		BypassCache: strings.EqualFold(c.GetHeader("X-Cache-Bypass"), "true"),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 0, "msg": err.Error()})
		return
	}
	for key, vals := range resp.Headers {
		for _, val := range vals {
			c.Header(key, val)
		}
	}
	c.Data(resp.StatusCode, resp.Headers.Get("Content-Type"), resp.Body)
}
