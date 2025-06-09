package http

import (
	"log"
	"microservice_swd_demo/service/customers/model/request"
	"microservice_swd_demo/service/customers/usecase"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type AccountController struct {
	accountUsecase usecase.ICustomerUseCase
}

func NewAccountController(accountUsecase usecase.ICustomerUseCase) *AccountController {
	return &AccountController{
		accountUsecase: accountUsecase,
	}
}

func (ac *AccountController) LoginHandler(c *gin.Context) {

	var loginRequest request.Login
	if err := c.ShouldBindJSON(&loginRequest); err != nil {
		log.Printf("Error binding JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	token, err := ac.accountUsecase.Login(loginRequest)
	if err != nil {
		log.Printf("Login failed: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}

func (ac *AccountController) RegisterHandler(c *gin.Context) {

	var registerRequest request.CreateAccountDTO
	if err := c.ShouldBindJSON(&registerRequest); err != nil {
		log.Printf("Error binding JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if _, err := ac.accountUsecase.CreateAccount(registerRequest); err != nil {
		log.Printf("Registration failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Registration failed"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "User registered successfully"})
}

func (ac *AccountController) GetAccountHandler(c *gin.Context) {
	// 1. Extract the customer ID from the path
	customerIdStr := c.Param("customerid")
	customerId, err := strconv.Atoi(customerIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid customer ID"})
		return
	}

	// 2. Call the usecase to get account info
	account, err := ac.accountUsecase.GetAccountByID(customerId)
	if err != nil {
		log.Printf("Error retrieving account: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve account"})
		return
	}

	// 3. Respond with the account info
	c.JSON(http.StatusOK, gin.H{"account": account})
}

func (ac *AccountController) RequestRide(c *gin.Context) {
	var rideRequest request.RideRequestDTO
	if err := c.ShouldBindJSON(&rideRequest); err != nil {
		log.Printf("Error binding JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := ac.accountUsecase.RequestRide(rideRequest); err != nil {
		log.Printf("Ride request failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to request ride"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Ride requested successfully, please wait for a driver to accept your request"})
}
