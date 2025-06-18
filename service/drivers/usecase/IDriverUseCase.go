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
	"strings"
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

					log.Printf("Processing ride request: %s for customer %d in region %s", ride.RequestId, ride.CustomerId, ride.RegionId)

					// Always delete the message after processing (even on failure)
					_, err := client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
						QueueUrl:      aws.String(queueURL),
						ReceiptHandle: msg.ReceiptHandle,
					})
					if err != nil {
						log.Printf("Delete failed: %v", err)
					} else {
						log.Printf("Message deleted: %s", ride.RequestId)
					}

					// 🚀 Process the ride request
					d.processRide(ride)

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
	log.Printf("Requesting driver %d to go online in region %d", request.DriverId, request.RegionId)

	err := d.redis.SAdd(context.Background(), CacheKeyPrefix+"online_drivers:"+strconv.Itoa(request.RegionId), request.DriverId)
	if err.Err() == nil {
		d.redis.Expire(context.Background(), CacheKeyPrefix+"online_drivers:"+strconv.Itoa(request.RegionId), 20*time.Minute)
	} else {
		log.Printf("Error adding driver %d to online list for region %d: %v", request.DriverId, request.RegionId, err.Err())
		return err.Err()
	}

	log.Printf("Driver %d successfully added to online list for region %d", request.DriverId, request.RegionId)
	return nil
}

func (d *driverUseCase) AcceptOrder(request request.AcceptOrderDTO) error {
	log.Printf("Starting to process accept order for request ID: %s and driver ID: %d", request.RequestId, request.DriverId)

	backOff := backoff.NewExponentialBackOff()
	backOff.MaxElapsedTime = 2 * time.Second // Set a maximum retry duration
	err := backoff.Retry(func() error {
		log.Printf("Attempting to publish accept order message for request ID: %s", request.RequestId)
		// Attempt to publish the message
		err := d.redis.Publish(context.Background(), CacheKeyPrefix+"accept_order:"+request.RequestId, request.DriverId)
		if err.Err() != nil {
			log.Printf("Error publishing accept order message for request ID: %s: %v", request.RequestId, err.Err())
			return err.Err()
		}
		log.Printf("Successfully published accept order message for request ID: %s", request.RequestId)
		return nil
	}, backOff)
	if err != nil {
		log.Printf("Failed to publish accept order message for request ID: %s after retries: %v", request.RequestId, err)
		return errors.New("failed to accept order")
	}
	log.Printf("Successfully processed accept order for request ID: %s and driver ID: %d", request.RequestId, request.DriverId)
	return nil
}

func (d *driverUseCase) processRide(ride RideRequest) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	var logBuilder strings.Builder
	defer func() {
		log.Print(logBuilder.String())
	}()

	logBuilder.WriteString(fmt.Sprintf("Processing ride request: %s for customer %d in region %s\n", ride.RequestId, ride.CustomerId, ride.RegionId))

	regionID := ride.RegionId
	redisKey := CacheKeyPrefix + "online_drivers:" + regionID

	// 1. Get online drivers from Redis
	logBuilder.WriteString(fmt.Sprintf("Fetching online drivers for region %s\n", regionID))
	drivers, err := d.redis.SMembers(ctx, redisKey).Result()
	if err != nil {
		logBuilder.WriteString(fmt.Sprintf("Failed to get drivers for region %s: %v\n", regionID, err))
		return
	}

	if len(drivers) == 0 {
		logBuilder.WriteString(fmt.Sprintf("No available drivers in region %s for ride %s\n", regionID, ride.RequestId))
		return
	}

	logBuilder.WriteString(fmt.Sprintf("Starting match for ride %s, %d drivers available\n", ride.RequestId, len(drivers)))

	userServiceURL := os.Getenv("USER_SERVICE_URL")
	if userServiceURL == "" {
		userServiceURL = "localhost:8081"
	}

	url := fmt.Sprintf("%s/customers/get-user-info/%d", userServiceURL, ride.CustomerId)
	logBuilder.WriteString(fmt.Sprintf("Fetching user info from URL: %s\n", url))
	account, err := d.fetchUserInfo(url)
	if err != nil {
		logBuilder.WriteString(fmt.Sprintf("Error fetching user info for CustomerID %d: %v\n", ride.CustomerId, err))
		return
	}

	for i, driverID := range drivers {
		select {
		case <-ctx.Done():
			logBuilder.WriteString(fmt.Sprintf("Global timeout reached for ride %s\n", ride.RequestId))
			return
		default:
			logBuilder.WriteString(fmt.Sprintf("[Unix Timestamp: %d] Notifying driver %s (attempt %d)\n", time.Now().Unix(), driverID, i+1))

			d.notifyDriver(driverID, ride, account)

			topic := CacheKeyPrefix + "accept_order:" + ride.RequestId
			logBuilder.WriteString(fmt.Sprintf("Subscribing to topic: %s\n", topic))
			sub := d.redis.Subscribe(ctx, topic)
			defer sub.Close()

			ch := sub.Channel()
			timer := time.NewTimer(30 * time.Second)
			defer timer.Stop()

			select {
			case msg := <-ch:
				logBuilder.WriteString(fmt.Sprintf("Received message from topic %s: %s\n", topic, msg.Payload))
				if msg.Payload == driverID {
					logBuilder.WriteString(fmt.Sprintf("Driver %s accepted the ride %s\n", driverID, ride.RequestId))
					driverid, _ := strconv.Atoi(driverID)
					driverInfo, err := d.driverRepo.GetByID(driverid)
					if err != nil {
						logBuilder.WriteString(fmt.Sprintf("Failed to fetch driver info: %v\n", err))
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

					logBuilder.WriteString(fmt.Sprintf("Sending NotifyUser message for ride %s to queue\n", ride.RequestId))
					_, err = queueClient.SendMessage(ctx, &sqs.SendMessageInput{
						QueueUrl:    aws.String(os.Getenv("USER_QUEUE_URL")),
						MessageBody: aws.String(string(payloadBytes)),
					})
					if err != nil {
						logBuilder.WriteString(fmt.Sprintf("Failed to publish NotifyUser message: %v\n", err))
					}
					return
				}
			case <-timer.C:
				logBuilder.WriteString(fmt.Sprintf("Driver %s did not respond, trying next\n", driverID))
			case <-ctx.Done():
				logBuilder.WriteString(fmt.Sprintf("Global timeout reached during driver wait for ride %s\n", ride.RequestId))
				return
			}
		}
	}

	logBuilder.WriteString(fmt.Sprintf("No driver accepted ride %s after checking all or timeout\n", ride.RequestId))
}

func (d *driverUseCase) notifyDriver(driverId string, request RideRequest, customer AccountResponseDTO) {
	ctx := context.Background()

	log.Printf("Starting notification process for driver %s and request %s", driverId, request.RequestId)

	// Create notification payload
	driverid, err := strconv.Atoi(driverId)
	if err != nil {
		log.Printf("Failed to convert driverId %s to integer: %v", driverId, err)
		return
	}

	log.Printf("Creating notification payload for driver %d", driverid)
	notifyPayload := dto.NotifyDriverDTO{
		DriverId:         driverid,
		RequestId:        request.RequestId,
		CustomerFullName: customer.FullName,
		CustomerPhone:    customer.Phone,
	}

	// Convert payload to JSON
	log.Printf("Marshalling notification payload to JSON")
	payloadBytes, err := json.Marshal(notifyPayload)
	if err != nil {
		log.Printf("Failed to marshal notify payload: %v", err)
		return
	}

	// Send message to driver queue
	queueURL := os.Getenv("DRIVER_QUEUE_URL")
	if queueURL == "" {
		log.Printf("DRIVER_QUEUE_URL environment variable is not set")
		return
	}

	log.Printf("Sending notification to driver queue at URL: %s", queueURL)
	_, err = queueClient.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    aws.String(queueURL),
		MessageBody: aws.String(string(payloadBytes)),
	})
	if err != nil {
		log.Printf("Failed to send notification to driver queue: %v", err)
		return
	}

	log.Printf("Notification successfully sent to driver %s for request %s", driverId, request.RequestId)
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
