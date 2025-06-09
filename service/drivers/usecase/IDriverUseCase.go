package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/cenkalti/backoff/v4"
	"github.com/redis/go-redis/v9"
	"log"
	"microservice_swd_demo/service/drivers/model/request"
	"microservice_swd_demo/service/drivers/model/response"
	"microservice_swd_demo/service/drivers/repository"
	"microservice_swd_demo/service/notify/model/dto"
	"net/http"
	"os"
	"strconv"
	"time"
)

type IDriverUseCase interface {
	GetDriverByID(driverID int) (response.DriverResponseDTO, error)
	RequestOnline(request request.OnlineRequestDTO) error
	AcceptOrder(request request.AcceptOrderDTO) error
	MatchingOrder() error
}

type NotifyUserDTO struct {
	RequestId string            `json:"request_id"`
	UserId    int               `json:"user_id"`
	Driver    DriverResponseDTO `json:"driver"`
}

type DriverResponseDTO struct {
	FullName string `json:"full_name"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	RegionID int    `json:"region_id"`
	Car      string `json:"car"`
}

type RideRequest struct {
	RequestId  string `json:"request_id"`
	CustomerId int    `json:"customer_id"`
	RegionId   string `json:"region_id"`
}

type AccountResponseDTO struct {
	CustomerID int    `json:"customer_id"`
	Email      string `json:"email"`
	Phone      string `json:"phone"`
	FullName   string `json:"full_name"`
	RegionID   int    `json:"region_id"`
}

const (
	CacheKeyPrefix = "driverservice:"
)

var (
	queueClient *sqs.Client
)

func init() {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("load config error: %v", err)
	}
	queueClient = sqs.NewFromConfig(cfg)
}

func NewDriverUseCase(driverRepo repository.IDriverRepository, redis *redis.Client) IDriverUseCase {
	return &driverUseCase{
		driverRepo: driverRepo,
		redis:      redis,
	}
}

type driverUseCase struct {
	driverRepo repository.IDriverRepository
	redis      *redis.Client
}

func (d *driverUseCase) MatchingOrder() error {

	ctx := context.Background()

	// Load AWS config
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("load config error: %v", err)
	}

	client := sqs.NewFromConfig(cfg)
	queueURL := os.Getenv("SQS_QUEUE_URL")

	go func() {
		for {
			output, err := client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
				QueueUrl:            aws.String(queueURL),
				MaxNumberOfMessages: 5,
				WaitTimeSeconds:     10,
				VisibilityTimeout:   30,
			})
			if err != nil {
				log.Printf("Receive error: %v", err)
				time.Sleep(5 * time.Second)
				continue
			}

			for _, msg := range output.Messages {
				go func(msg types.Message) {
					var ride RideRequest
					if err := json.Unmarshal([]byte(*msg.Body), &ride); err != nil {
						log.Printf("Invalid message: %v", err)
						return
					}

					// 🚀 Process the ride request
					d.processRide(ride)

					// ✅ Delete message after successful processing
					_, err := client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
						QueueUrl:      aws.String(queueURL),
						ReceiptHandle: msg.ReceiptHandle,
					})
					if err != nil {
						log.Printf("Delete failed: %v", err)
					} else {
						log.Printf("Message deleted: %s", ride.RequestId)
					}
				}(msg)
			}
		}
	}()

	return nil
}

func (d *driverUseCase) GetDriverByID(driverID int) (response.DriverResponseDTO, error) {

	driver, err := d.driverRepo.GetByID(driverID)
	if err != nil {
		return response.DriverResponseDTO{}, err
	}

	return response.DriverResponseDTO{
		FullName: driver.Name,
		Email:    driver.Email,
		Phone:    driver.Phone,
		RegionID: driver.RegionID,
		Car:      driver.Car,
	}, nil
}

func (d *driverUseCase) RequestOnline(request request.OnlineRequestDTO) error {
	err := d.redis.SAdd(context.Background(), CacheKeyPrefix+"online_drivers:"+strconv.Itoa(request.RegionId), request.DriverId)
	if err.Err() != nil {
		log.Printf("Error adding driver to online list: %v", err.Err())
		return errors.New("failed to add driver to online list: ")
	}

	return nil
}

func (d *driverUseCase) AcceptOrder(request request.AcceptOrderDTO) error {

	backOff := backoff.NewExponentialBackOff()
	backOff.MaxElapsedTime = 2 * time.Second // Set a maximum retry duration
	err := backoff.Retry(func() error {
		// Attempt to publish the message
		err := d.redis.Publish(context.Background(), CacheKeyPrefix+"accept_order:"+request.RequestId, request.DriverId)
		if err.Err() != nil {
			log.Printf("Error publishing accept order message: %v", err.Err())
			return err.Err()
		}
		return nil
	}, backOff)
	if err != nil {
		log.Printf("Failed to publish accept order message after retries: %v", err)
		return errors.New("failed to accept order")
	}

	return nil
}

func (d *driverUseCase) processRide(ride RideRequest) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	regionID := ride.RegionId
	redisKey := CacheKeyPrefix + "online_drivers:" + regionID

	// 1. Get online drivers from Redis
	drivers, err := d.redis.SMembers(ctx, redisKey).Result()
	if err != nil {
		log.Printf("Failed to get drivers for region %s: %v", regionID, err)
		return
	}

	if len(drivers) == 0 {
		log.Printf("No available drivers in region %s for ride %s", regionID, ride.RequestId)
		return
	}

	log.Printf("Starting match for ride %s, %d drivers available", ride.RequestId, len(drivers))

	userServiceURL := os.Getenv("USER_SERVICE_URL")
	if userServiceURL == "" {
		log.Printf("USER_SERVICE_URL is not set")
		return
	}

	url := fmt.Sprintf("%s/get-user-info/%d", userServiceURL, ride.CustomerId)
	account, err := d.fetchUserInfo(url)
	if err != nil {
		log.Printf("Error fetching user info for CustomerID %d: %v", ride.CustomerId, err)
		return
	}

	for i, driverID := range drivers {
		select {
		case <-ctx.Done():
			log.Printf("Global timeout reached for ride %s", ride.RequestId)
			return
		default:
			log.Printf("Notifying driver %s (attempt %d)", driverID, i+1)

			d.notifyDriver(driverID, ride, account)

			topic := CacheKeyPrefix + "accept_order:" + ride.RequestId
			sub := d.redis.Subscribe(ctx, topic)
			defer sub.Close()

			ch := sub.Channel()
			timer := time.NewTimer(8 * time.Second)
			defer timer.Stop()

			select {
			case msg := <-ch:
				if msg.Payload == driverID {
					driverid, _ := strconv.Atoi(driverID)
					driverInfo, err := d.driverRepo.GetByID(driverid)
					if err != nil {
						log.Printf("Failed to fetch driver info: %v", err)
						return
					}

					notifyPayload := NotifyUserDTO{
						RequestId: ride.RequestId,
						UserId:    ride.CustomerId,
						Driver: DriverResponseDTO{
							FullName: driverInfo.Name,
							Email:    driverInfo.Email,
							Phone:    driverInfo.Phone,
							RegionID: driverInfo.RegionID,
							Car:      driverInfo.Car,
						},
					}

					payloadBytes, _ := json.Marshal(notifyPayload)

					_, err = queueClient.SendMessage(ctx, &sqs.SendMessageInput{
						QueueUrl:    aws.String(os.Getenv("USER_QUEUE_URL")),
						MessageBody: aws.String(string(payloadBytes)),
					})
					if err != nil {
						log.Printf("Failed to publish NotifyUser message: %v", err)
					}
					return
				}
			case <-timer.C:
				log.Printf("Driver %s did not respond, trying next", driverID)
			case <-ctx.Done():
				log.Printf("Global timeout reached during driver wait for ride %s", ride.RequestId)
				return
			}
		}
	}

	log.Printf("No driver accepted ride %s after checking all or timeout", ride.RequestId)
}

func (d *driverUseCase) notifyDriver(driverId string, request RideRequest, customer AccountResponseDTO) {
	ctx := context.Background()

	// Create notification payload
	driverid, _ := strconv.Atoi(driverId)

	notifyPayload := dto.NotifyDriverDTO{
		DriverId:         driverid,
		RequestId:        request.RequestId,
		CustomerFullName: customer.FullName,
		CustomerPhone:    customer.Phone,
	}

	// Convert payload to JSON
	payloadBytes, err := json.Marshal(notifyPayload)
	if err != nil {
		log.Printf("Failed to marshal notify payload: %v", err)
		return
	}

	// Send message to driver queue
	_, err = queueClient.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    aws.String(os.Getenv("DRIVER_QUEUE_URL")),
		MessageBody: aws.String(string(payloadBytes)),
	})
	if err != nil {
		log.Printf("Failed to send notification to driver queue: %v", err)
		return
	}

	log.Printf("Notification sent to driver %s for request %s", driverId, request.RequestId)
}

func (d *driverUseCase) fetchUserInfo(url string) (AccountResponseDTO, error) {
	resp, err := http.Get(url)
	if err != nil {
		return AccountResponseDTO{}, fmt.Errorf("failed to fetch user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return AccountResponseDTO{}, fmt.Errorf("user service responded with status: %d", resp.StatusCode)
	}

	var result struct {
		Account AccountResponseDTO `json:"account"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return AccountResponseDTO{}, fmt.Errorf("failed to decode user info: %w", err)
	}

	return result.Account, nil
}
