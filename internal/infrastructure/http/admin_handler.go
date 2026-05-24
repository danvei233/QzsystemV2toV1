package httpadapter

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"xiaoheiproxy/internal/app/admin"
	"xiaoheiproxy/internal/app/gateway"
	"xiaoheiproxy/internal/domain"
	"xiaoheiproxy/internal/infrastructure/config"

	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	service    *admin.Service
	gateway    *gateway.Service
	cfg        *config.Config
	configPath string
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type updateSettingsRequest struct {
	AdminPassword    string   `json:"admin_password"`
	UpstreamBaseURL  string   `json:"upstream_base_url" binding:"required,url"`
	UpstreamAPIKey   string   `json:"upstream_api_key"`
	Timeout          string   `json:"timeout" binding:"required"`
	CacheTTL         string   `json:"cache_ttl" binding:"required"`
	MaxBodyBytes     int64    `json:"max_body_bytes" binding:"gte=1024,lte=10485760"`
	RequireAPIKey    bool     `json:"require_api_key"`
	AcceptedAPIKeys  []string `json:"accepted_api_keys"`
	LogRetentionDays int      `json:"log_retention_days" binding:"gte=1,lte=3650"`
	LogMaxSizeMB     int64    `json:"log_max_size_mb" binding:"gte=1,lte=102400"`
}

func NewAdminHandler(service *admin.Service, gatewaySvc *gateway.Service, cfg *config.Config, configPath string) *AdminHandler {
	return &AdminHandler{service: service, gateway: gatewaySvc, cfg: cfg, configPath: configPath}
}

func (h *AdminHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username/password required"})
		return
	}
	token, expiresAt, err := h.service.Login(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token, "expires_at": expiresAt})
}

func (h *AdminHandler) Logout(c *gin.Context) {
	h.service.Logout(bearerToken(c))
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *AdminHandler) Me(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *AdminHandler) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !h.service.Validate(bearerToken(c)) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func (h *AdminHandler) ListLogs(c *gin.Context) {
	filter := domain.RequestLogFilter{
		Keyword: strings.TrimSpace(c.Query("keyword")),
		Path:    strings.TrimSpace(c.Query("path")),
		Limit:   intQuery(c, "limit", 50),
		Offset:  intQuery(c, "offset", 0),
	}
	if raw := strings.TrimSpace(c.Query("success")); raw != "" {
		v := raw == "1" || strings.EqualFold(raw, "true")
		filter.Success = &v
	}
	items, total, err := h.service.ListLogs(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
}

func (h *AdminHandler) GetLog(c *gin.Context) {
	id64, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	item, err := h.service.GetLog(c.Request.Context(), uint(id64))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *AdminHandler) Settings(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"server_addr":          h.cfg.Server.Addr,
		"admin_username":       h.cfg.Admin.Username,
		"upstream_base_url":    h.cfg.Gateway.UpstreamBaseURL,
		"upstream_api_key_set": h.cfg.Gateway.UpstreamAPIKey != "",
		"timeout":              h.cfg.Gateway.Timeout.String(),
		"cache_ttl":            h.cfg.Gateway.CacheTTL.String(),
		"max_body_bytes":       h.cfg.Gateway.MaxBodyBytes,
		"require_api_key":      h.cfg.Gateway.RequireAPIKey,
		"accepted_api_keys":    h.cfg.Gateway.AcceptedAPIKeys,
		"database_dsn":         h.cfg.Database.DSN,
		"log_retention_days":   h.cfg.Logs.Request.RetentionDays,
		"log_max_size_mb":      h.cfg.Logs.Request.MaxSizeMB,
	})
}

func (h *AdminHandler) UpdateSettings(c *gin.Context) {
	var req updateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	timeout, err := time.ParseDuration(req.Timeout)
	if err != nil || timeout <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid timeout"})
		return
	}
	cacheTTL, err := time.ParseDuration(req.CacheTTL)
	if err != nil || cacheTTL <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid cache_ttl"})
		return
	}
	apiKey := strings.TrimSpace(req.UpstreamAPIKey)
	if apiKey == "" {
		apiKey = h.cfg.Gateway.UpstreamAPIKey
	}
	if apiKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "upstream_api_key required"})
		return
	}
	keys := compactStrings(req.AcceptedAPIKeys)
	if req.RequireAPIKey && len(keys) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "accepted_api_keys required when require_api_key is enabled"})
		return
	}
	h.cfg.Gateway.UpstreamBaseURL = strings.TrimSpace(req.UpstreamBaseURL)
	h.cfg.Gateway.UpstreamAPIKey = apiKey
	h.cfg.Gateway.Timeout = timeout
	h.cfg.Gateway.CacheTTL = cacheTTL
	h.cfg.Gateway.MaxBodyBytes = req.MaxBodyBytes
	h.cfg.Gateway.RequireAPIKey = req.RequireAPIKey
	h.cfg.Gateway.AcceptedAPIKeys = keys
	h.cfg.Logs.Request.RetentionDays = req.LogRetentionDays
	h.cfg.Logs.Request.MaxSizeMB = req.LogMaxSizeMB
	if strings.TrimSpace(req.AdminPassword) != "" {
		h.cfg.Admin.Password = strings.TrimSpace(req.AdminPassword)
		h.service.UpdatePassword(h.cfg.Admin.Password)
	}
	if err := h.cfg.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := config.Save(h.configPath, h.cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.gateway.UpdateConfig(gateway.RuntimeConfig{
		UpstreamBaseURL: h.cfg.Gateway.UpstreamBaseURL,
		UpstreamAPIKey:  h.cfg.Gateway.UpstreamAPIKey,
		Timeout:         h.cfg.Gateway.Timeout,
		CacheTTL:        h.cfg.Gateway.CacheTTL,
		MaxBodyBytes:    h.cfg.Gateway.MaxBodyBytes,
		RedactFields:    h.cfg.Gateway.RedactFields,
	})
	if err := h.service.UpdateLogRetention(domain.LogRetentionPolicy{
		RetentionDays: h.cfg.Logs.Request.RetentionDays,
		MaxSizeBytes:  h.cfg.Logs.Request.MaxSizeMB * 1024 * 1024,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func bearerToken(c *gin.Context) string {
	header := strings.TrimSpace(c.GetHeader("Authorization"))
	if strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return strings.TrimSpace(header[7:])
	}
	return ""
}

func intQuery(c *gin.Context, key string, fallback int) int {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return v
}

func compactStrings(items []string) []string {
	out := make([]string, 0, len(items))
	seen := map[string]struct{}{}
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}
