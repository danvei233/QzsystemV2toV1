package httpadapter

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"xiaoheiproxy/internal/app/admin"
	"xiaoheiproxy/internal/app/gateway"
	"xiaoheiproxy/internal/infrastructure/config"

	"github.com/gin-gonic/gin"
)

//go:embed admin_dist/*
var embeddedAdminDist embed.FS

type RouterDeps struct {
	Config     *config.Config
	ConfigPath string
	Gateway    *gateway.Service
	Admin      *admin.Service
	Version    string
	Commit     string
	BuildTime  string
}

func NewRouter(deps RouterDeps) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery(), traceMiddleware())
	if len(deps.Config.Server.TrustedProxies) > 0 {
		_ = r.SetTrustedProxies(deps.Config.Server.TrustedProxies)
	}

	adminHandler := NewAdminHandler(deps.Admin, deps.Gateway, deps.Config, deps.ConfigPath)
	gatewayHandler := NewGatewayHandler(deps.Gateway, &deps.Config.Gateway)

	api := r.Group("/admin/api")
	api.POST("/login", adminHandler.Login)
	api.POST("/logout", adminHandler.RequireAuth(), adminHandler.Logout)
	api.GET("/me", adminHandler.RequireAuth(), adminHandler.Me)
	api.GET("/logs", adminHandler.RequireAuth(), adminHandler.ListLogs)
	api.GET("/logs/:id", adminHandler.RequireAuth(), adminHandler.GetLog)
	api.GET("/stats/error-rates", adminHandler.RequireAuth(), adminHandler.EndpointErrorStats)
	api.GET("/settings", adminHandler.RequireAuth(), adminHandler.Settings)
	api.PATCH("/settings", adminHandler.RequireAuth(), adminHandler.UpdateSettings)

	v1 := r.Group("/api/v1")
	v1.Use(gatewayHandler.RequireClientKey())
	v1.POST("/*action", gatewayHandler.Handle)
	v1.GET("/*action", gatewayHandler.Handle)

	mountAdminStatic(r)
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"ok":         true,
			"version":    deps.Version,
			"commit":     deps.Commit,
			"build_time": deps.BuildTime,
		})
	})
	return r
}

func mountAdminStatic(r *gin.Engine) {
	if distFS, err := fs.Sub(embeddedAdminDist, "admin_dist"); err == nil {
		if assetsFS, err := fs.Sub(distFS, "assets"); err == nil {
			r.StaticFS("/admin/assets", http.FS(assetsFS))
		}
		r.GET("/admin/", func(c *gin.Context) {
			serveEmbeddedIndex(c, distFS)
		})
		r.NoRoute(func(c *gin.Context) {
			if strings.HasPrefix(c.Request.URL.Path, "/admin") {
				serveEmbeddedIndex(c, distFS)
				return
			}
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		})
		return
	}
	dist := filepath.Join("web", "dist")
	index := filepath.Join(dist, "index.html")
	if _, err := os.Stat(index); err == nil {
		r.Static("/admin/assets", filepath.Join(dist, "assets"))
		r.StaticFile("/admin/", index)
		r.NoRoute(func(c *gin.Context) {
			if strings.HasPrefix(c.Request.URL.Path, "/admin") {
				c.File(index)
				return
			}
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		})
		return
	}
	r.GET("/admin/", func(c *gin.Context) {
		c.String(http.StatusOK, "admin UI is not built yet. Run: cd web && npm install && npm run build")
	})
}

func serveEmbeddedIndex(c *gin.Context, distFS fs.FS) {
	b, err := fs.ReadFile(distFS, "index.html")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "admin UI index not found"})
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", b)
}
