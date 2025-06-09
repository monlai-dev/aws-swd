package http

import (
	"log"
	"microservice_swd_demo/service/notify/model/dto"
	"microservice_swd_demo/service/notify/usecase"
	"net/http"

	"github.com/gin-gonic/gin"
)

type NotifyController struct {
	useCase usecase.INotifyUseCase
}

func NewNotifyController(useCase usecase.INotifyUseCase) *NotifyController {
	return &NotifyController{
		useCase: useCase,
	}
}

func (n *NotifyController) RegisterFcmTokenHandler(c *gin.Context) {
	var tokenDTO dto.RequestUpdateFcmTokenDTO
	if err := c.ShouldBindJSON(&tokenDTO); err != nil {
		log.Printf("Failed to bind FCM token request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := n.useCase.RegisterFcmToken(tokenDTO); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register FCM token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "FCM token registered successfully"})
}
