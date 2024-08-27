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
func router(g *gin.Engine) {
	// Ping test
	g.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	// News API
	s := storage.NewNewsService()
	g.GET("/home/articles/token/:token", utils.ApiHandle(s.HomeLinkHandler))
	g.POST("/home/news", utils.ApiHandle(s.HomeHandler))
	g.POST("/home/news/sitemap/:category/:lang", utils.ApiHandle(s.HomeListHandler))
	g.POST("/home/categorylist", utils.ApiHandle(s.HomeOriginListHandler))
	g.POST("/news/readlist", utils.ApiHandle(s.NewsReadListHandler))
	g.POST("/news/:origin", utils.ApiHandle(s.NewsOriginListHandler))
	g.POST("/news/search", utils.ApiHandle(s.NewsSearchHandler))
}

func StartAPIServer() {
	gin.SetMode(config.Cfg.API.Mode)

	g := gin.New()
	g.Use(gin.Logger(), gin.Recovery())
	router(g)

	svc := http.Server{
		Addr:           config.Cfg.API.Addr,
		Handler:        g,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	if err := svc.ListenAndServe(); err != nil {
		logger.Infof("Failed to start API server: %v", err)
	}
}
