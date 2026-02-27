package api

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"qq-farm-bot/internal/auth"
	"qq-farm-bot/internal/bot"
	"qq-farm-bot/internal/config"
	"qq-farm-bot/internal/store"
)

func SetupRouter(cfg *config.Config, s *store.Store, mgr *bot.Manager, frontendFS fs.FS) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.SetTrustedProxies(nil)

	// CORS
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Public routes
	api := r.Group("/api")
	auth.RegisterRoutes(api.Group("/auth"), cfg, s)

	// Protected routes
	protected := api.Group("")
	protected.Use(auth.AuthMiddleware(cfg.JWTSecret))
	{
		RegisterAccountRoutes(protected, s, mgr, cfg)
		RegisterBotRoutes(protected, s, mgr)
		RegisterLogRoutes(protected, s, mgr)
		RegisterDashboardRoutes(protected, s, mgr)
	}

	// Serve frontend static files from embedded FS
	if frontendFS != nil {
		httpFS := http.FS(frontendFS)
		r.StaticFS("/assets", &onlyFilesFS{httpFS})

		// Pre-read index.html to avoid http.FileServer's automatic
		// "/index.html" -> "/" redirect which causes infinite redirect loops.
		indexHTML, _ := fs.ReadFile(frontendFS, "index.html")

		r.NoRoute(func(c *gin.Context) {
			// API routes return 404 JSON
			if strings.HasPrefix(c.Request.URL.Path, "/api") {
				c.JSON(404, gin.H{"error": "not found"})
				return
			}
			// Try to serve static file first (skip "/" to avoid redirect loop)
			path := c.Request.URL.Path
			if path != "/" {
				f, err := frontendFS.Open(strings.TrimPrefix(path, "/"))
				if err == nil {
					f.Close()
					c.FileFromFS(path, httpFS)
					return
				}
			}
			// SPA fallback: serve index.html directly (bypasses http.FileServer redirect)
			c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
		})
	}

	return r
}

// onlyFilesFS wraps http.FileSystem to disable directory listings
type onlyFilesFS struct {
	fs http.FileSystem
}

func (o *onlyFilesFS) Open(name string) (http.File, error) {
	f, err := o.fs.Open(name)
	if err != nil {
		return nil, err
	}
	stat, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	if stat.IsDir() {
		f.Close()
		return nil, fs.ErrNotExist
	}
	return f, nil
}
