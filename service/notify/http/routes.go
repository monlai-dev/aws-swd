package http

import "github.com/gin-gonic/gin"

func RegisterNotifyRoutes(engine *gin.Engine, controller *NotifyController) {
	group := engine.Group("/notify")
	group.POST("/register-fcm-token", controller.RegisterFcmTokenHandler)
}
