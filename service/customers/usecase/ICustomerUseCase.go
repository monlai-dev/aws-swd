package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"log"
	"microservice_swd_demo/service/customers/helper"
	"microservice_swd_demo/service/customers/model/postgres"
	"microservice_swd_demo/service/customers/model/request"
	"microservice_swd_demo/service/customers/model/response"
	"microservice_swd_demo/service/customers/repository"
	"os"
	"strconv"
	"time"
)

type ICustomerUseCase interface {
	CreateAccount(dto request.CreateAccountDTO) (string, error)
	Login(dto request.Login) (response.LoginResponseDTO, error)
	GetAccountByID(id int) (response.AccountResponseDTO, error)
	RequestRide(request request.RideRequestDTO) error
}

const (
	CacheKeyPrefix = "customerservice:"
)

type RideRequest struct {
	RequestId  string `json:"request_id"`
	CustomerId int    `json:"customer_id"`
	RegionId   string `json:"region_id"`
}

type customerUseCase struct {
	customerRepo repository.ICustomerRepository
	redisClient  *redis.Client
}

func NewCustomerUseCase(customerRepo repository.ICustomerRepository, redisclient *redis.Client) ICustomerUseCase {
	return &customerUseCase{
		customerRepo: customerRepo,
		redisClient:  redisclient,
	}
}

func (c customerUseCase) CreateAccount(dto request.CreateAccountDTO) (string, error) {

	hashedPassword, _ := helper.HashPassword(dto.Password)

	postgresAccount := postgres.Customer{
		Name:     dto.FullName,
		Email:    dto.Email,
		Password: hashedPassword,
		Phone:    dto.Phone,
		RegionID: dto.RegionID,
	}

	err := c.customerRepo.CreateAccount(postgresAccount)
	if err != nil {
		log.Printf("Error creating account: %s", err.Error())
		return "", errors.New("failed to create account: ")
	}

	return "Account created success", nil

}

func (c customerUseCase) Login(dto request.Login) (response.LoginResponseDTO, error) {

	postgresAccount, err := c.customerRepo.GetByUsernameAndPassword(dto.Email)
	if err != nil {
		log.Printf("Error fetching account: %s", err.Error())
		return response.LoginResponseDTO{}, errors.New("failed to login: ")
	}

	if err := helper.ComparePasswords(postgresAccount.Password, dto.Password); err != nil {
		log.Printf("Password mismatch for account: %s", postgresAccount.Email)
		return response.LoginResponseDTO{}, errors.New("invalid email or password")
	}

	token, err := helper.CreateToken(postgresAccount.Email, int(postgresAccount.ID), "customer")
	if err != nil {
		return response.LoginResponseDTO{}, errors.New("failed to create token: ")
	}

	loginResponse := response.LoginResponseDTO{
		AccessToken: token,
		UserId:      int(postgresAccount.ID),
	}

	return loginResponse, nil

}

func (c customerUseCase) GetAccountByID(id int) (response.AccountResponseDTO, error) {
	cacheKey := CacheKeyPrefix + "customer" + strconv.Itoa(id)

	value := c.redisClient.Get(context.Background(), cacheKey)

	if value.Err() == nil {
		var cachedAccount response.AccountResponseDTO
		if err := json.Unmarshal([]byte(value.Val()), &cachedAccount); err != nil {
			log.Printf("Error scanning cached account: %s", err.Error())
			return response.AccountResponseDTO{}, errors.New("error retrieving cached account")
		}
		return cachedAccount, nil
	}

	postgresAccount, err := c.customerRepo.GetByID(id)

	if errors.Is(err, gorm.ErrRecordNotFound) {
		log.Printf("Error fetching account by ID: %s", err.Error())
		return response.AccountResponseDTO{}, errors.New("account not found for the given ID")
	}

	if err != nil {
		log.Printf("Error fetching account by ID: %s", err.Error())
		return response.AccountResponseDTO{}, errors.New("something went wrong while fetching account")
	}

	responseAccount := response.AccountResponseDTO{
		CustomerID: int(postgresAccount.ID),
		FullName:   postgresAccount.Name,
		Email:      postgresAccount.Email,
		Phone:      postgresAccount.Phone,
		RegionID:   postgresAccount.RegionID,
	}

	go func() {
		// Store as JSON
		jsonData, err := json.Marshal(responseAccount)
		if err != nil {
			log.Printf("Failed to marshal account: %s", err)
			return
		}

		if err := c.redisClient.Set(context.Background(), cacheKey, jsonData, 5*time.Minute).Err(); err != nil {
			log.Printf("Failed to write cache for customer %d: %s", id, err)
		} else {
			log.Printf("Cached customer %d in Redis", id)
		}
	}()

	return responseAccount, nil
}

func (c customerUseCase) RequestRide(request request.RideRequestDTO) error {
	ctx := context.Background()

	// Load AWS config
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("load config error: %v", err)
	}

	client := sqs.NewFromConfig(cfg)
	queueURL := os.Getenv("SQS_QUEUE_URL")

	randomRequestID := uuid.New().String()

	rideRequest := RideRequest{
		RequestId:  randomRequestID,
		CustomerId: request.UserId,
		RegionId:   request.RegionId,
	}

	data, err := json.Marshal(rideRequest)
	if err != nil {
		log.Printf("Failed to marshal ride request: %s", err.Error())
		return errors.New("failed to create ride request")
	}

	_, err = client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    &queueURL,
		MessageBody: aws.String(string(data)),
	})
	if err != nil {
		log.Printf("Failed to send message: %v", err)
		return errors.New("failed to enqueue ride request")
	}

	log.Printf("Ride request sent with ID: %s", randomRequestID)
	return nil
}
