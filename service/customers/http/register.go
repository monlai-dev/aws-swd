package http

import "github.com/gin-gonic/gin"

func RegisterCustomerRoutes(engine *gin.Engine, controller *AccountController) {
	group := engine.Group("/customers")
	group.POST("/login", controller.LoginHandler)
	group.POST("/register", controller.RegisterHandler)
	group.GET("/get-user-info/:customerid", controller.GetAccountHandler)
}
