package cmd

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"log"
)

type ServerConfig struct {
	Port int
}

func NewGinEngine() *gin.Engine {
	engine := gin.Default()
	engine.Use(gin.Recovery())
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
