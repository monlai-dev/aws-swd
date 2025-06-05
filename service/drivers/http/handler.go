package http

import (
	"github.com/gin-gonic/gin"
	"log"
	"microservice_swd_demo/service/drivers/model/request"
	"microservice_swd_demo/service/drivers/usecase"
	"net/http"
	"strconv"
)

type DriversHandler struct {
	iDriverUseCase usecase.IDriverUseCase
}

func NewDriversHandler(iDriverUseCase usecase.IDriverUseCase) *DriversHandler {
	return &DriversHandler{
		iDriverUseCase: iDriverUseCase,
	}
}

func (h *DriversHandler) RequestOnline(c *gin.Context) {

	var onlineRequest request.OnlineRequestDTO
	if err := c.ShouldBindJSON(&onlineRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid onlineRequest data",
		})
		return
	}

	if err := h.iDriverUseCase.RequestOnline(onlineRequest); err != nil {
		log.Printf("Error processing online onlineRequest: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to process onlineRequest",
			"details": "Please try again later or contact support",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Online onlineRequest processed successfully",
	})
}

func (h *DriversHandler) AcceptOrder(c *gin.Context) {

	var acceptOrderRequest request.AcceptOrderDTO
	if err := c.ShouldBindJSON(&acceptOrderRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid acceptOrderRequest data",
		})
		return
	}

	if err := h.iDriverUseCase.AcceptOrder(acceptOrderRequest); err != nil {
		log.Printf("Error processing accept order request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to process accept order request",
			"details": "Please try again later or contact support",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Order accepted successfully",
	})

}

func (h *DriversHandler) GetDriverInfo(c *gin.Context) {

	driverID, _ := strconv.Atoi(c.Param("driver_id"))
	if driverID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Driver ID is required",
		})
		return
	}

	driverInfo, err := h.iDriverUseCase.GetDriverByID(driverID)
	if err != nil {
		log.Printf("Error fetching driver info: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch driver info",
			"details": "Please try again later or contact support",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"driver_info": driverInfo,
	})
}
