package cmd

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/fx"
	"log"
)

type ServerConfig struct {
	Port int
}

func NewGinEngine() *gin.Engine {
	engine := gin.Default()
	engine.Use(gin.Recovery())
	engine.Use(CORSMiddleware())

	engine.GET("/metrics", gin.WrapH(promhttp.Handler()))
	return engine
}

func providePort(port int) func() ServerConfig {
	return func() ServerConfig {
		return ServerConfig{Port: port}
	}
}

func StartHTTPServer(lc fx.Lifecycle, engine *gin.Engine, cfg ServerConfig) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				addr := fmt.Sprintf(":%d", cfg.Port)
				log.Printf("Starting HTTP server at %s", addr)
				if err := engine.Run(addr); err != nil {
					log.Fatalf("Failed to start server: %v", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Println("Stopping HTTP server")
			return nil
		},
	})
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		//if c.Request.Method == "OPTIONS" {
		//	c.AbortWithStatus(204)
		//	return
		//}
		c.Next()
	}
}
