package cmd

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"news/src/config"
	"news/src/logger"
	"news/src/storage"
	"news/src/utils"
	"time"
)

// router sets up the API routes.
func router(g *gin.Engine, ns *storage.NewsService) {
	// Ping test
	g.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	// News API
	g.GET("/news/articles/token/:token", utils.ApiHandle(ns.HomeLinkHandler))
	g.POST("/news/home", utils.ApiHandle(ns.HomeHandler))
	g.POST("/news/sitemap/:category/:lang", utils.ApiHandle(ns.HomeListHandler))
	g.POST("/news/origins", utils.ApiHandle(ns.HomeOriginListHandler))
	g.POST("/news/reads", utils.ApiHandle(ns.NewsReadListHandler))
	g.POST("/news/:origin", utils.ApiHandle(ns.NewsOriginListHandler))
	g.POST("/news/search", utils.ApiHandle(ns.NewsSearchHandler))
}

func StartAPIServer() {
	gin.SetMode(config.Cfg.API.Mode)

	// Initialize news service
	ns := storage.NewNewsService()
	defer ns.Release()

	g := gin.New()
	g.Use(gin.Logger(), gin.Recovery())
	router(g, ns)

	svc := http.Server{
		Addr:           config.Cfg.API.Addr,
		Handler:        g,
		MaxHeaderBytes: 1 << 20,
	}
	if config.Cfg.API.Mode == gin.ReleaseMode {
		svc.ReadTimeout = 15 * time.Second
		svc.WriteTimeout = 15 * time.Second
	}

	if err := svc.ListenAndServe(); err != nil {
		logger.Infof("Failed to start API server: %v", err)
	}
}
