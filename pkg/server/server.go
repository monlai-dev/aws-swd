package server

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type HTTPServer struct {
	server *http.Server
	router *gin.Engine
}

func NewServer() *HTTPServer {
	router := gin.Default()
	
	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	return &HTTPServer{
		server: server,
		router: router,
	}
}

func (s *HTTPServer) Start() error {
	// Register routes
	s.registerRoutes()
	
	// Start server
	return s.server.ListenAndServe()
}

func (s *HTTPServer) Stop(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return s.server.Shutdown(ctx)
}

func (s *HTTPServer) registerRoutes() {
	// Health check
	s.router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	// Add more routes here
} 