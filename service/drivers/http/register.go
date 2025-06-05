package http

import "github.com/gin-gonic/gin"

func RegisterRoutes(engine *gin.Engine, handler *DriversHandler) {
	group := engine.Group("/drivers")
	group.POST("/request-online", handler.RequestOnline)
	group.POST("/accept-order", handler.AcceptOrder)
	group.GET("/info", handler.GetDriverInfo)
}
