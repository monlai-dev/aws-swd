package http

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func RegisterNotifyRoutes(engine *gin.Engine, controller *NotifyController) {
	group := engine.Group("/notify")
	group.POST("/register-fcm-token", controller.RegisterFcmTokenHandler)
	group.GET("/metrics", gin.WrapH(promhttp.Handler()))
}
