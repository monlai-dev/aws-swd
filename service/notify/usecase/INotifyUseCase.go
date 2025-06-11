package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"microservice_swd_demo/service/notify/model/dto"
	"microservice_swd_demo/service/notify/strategy"
	"os"
	"strconv"
	"time"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/redis/go-redis/v9"
)

var (
	queueClient *sqs.Client
	notifyChan  = make(chan dto.NotifyDriverDTO, 100)
)

func init() {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("load config error: %v", err)
	}
	queueClient = sqs.NewFromConfig(cfg)
}

type INotifyUseCase interface {
	NotifyDriver() error
	RegisterFcmToken(dto dto.RequestUpdateFcmTokenDTO) error
	NotifyUser() error
	GetFcmToken(role string, accountId int) (string, error)
}

type notifyUseCase struct {
	redis          *redis.Client
	firebaseClient *firebase.App
}

func NewNotifyUseCase(redis *redis.Client, firebaseClient *firebase.App) INotifyUseCase {
	return &notifyUseCase{
		redis:          redis,
		firebaseClient: firebaseClient,
	}
}

func (n *notifyUseCase) NotifyDriver() error {
	log.Println("Starting NotifyDriver process...")
	queueUrl := os.Getenv("DRIVER_QUEUE_URL")
	log.Printf("Driver queue URL: %s", queueUrl)

	go func() { // Start processing in a goroutine
		for {

			output, err := queueClient.ReceiveMessage(context.Background(), &sqs.ReceiveMessageInput{
				QueueUrl:            &queueUrl,
				MaxNumberOfMessages: 5,
				WaitTimeSeconds:     2,
			})
			if err != nil {
				log.Printf("Failed to receive message: %v", err)
				time.Sleep(5 * time.Second)
				continue
			}

			for _, msg := range output.Messages {
				log.Printf("Processing message with ID: %s", *msg.MessageId)
				var driverDTO dto.NotifyDriverDTO
				err := json.Unmarshal([]byte(*msg.Body), &driverDTO)
				if err != nil {
					log.Printf("Failed to parse message body: %v", err)
					continue
				}

				log.Printf("Enqueuing message for driver ID: %d", driverDTO.DriverId)
				notifyChan <- driverDTO // Push to worker channel

				// Delete message after enqueue
				log.Printf("Deleting message with ID: %s", *msg.MessageId)
				_, err = queueClient.DeleteMessage(context.Background(), &sqs.DeleteMessageInput{
					QueueUrl:      &queueUrl,
					ReceiptHandle: msg.ReceiptHandle,
				})
				if err != nil {
					log.Printf("Failed to delete message: %v", err)
				} else {
					log.Printf("Successfully deleted message with ID: %s", *msg.MessageId)
				}
			}
		}
	}()

	log.Println("NotifyDriver process started successfully.")
	return nil
}

func (n *notifyUseCase) RegisterFcmToken(dto dto.RequestUpdateFcmTokenDTO) error {
	ctx := context.Background()
	strat, err := strategy.GetTokenStrategy(dto.Role)
	if err != nil {
		log.Printf("Invalid role: %v", err)
		return err
	}
	if err := strat.StoreToken(ctx, n.redis, dto); err != nil {
		log.Printf("Failed to store token: %v", err)
		return err
	}
	log.Printf("Stored FCM token for user %d with role %s", dto.UserId, dto.Role)
	return nil
}

func (n *notifyUseCase) NotifyUser() error {
	log.Println("Starting NotifyUser process...")
	queueUrl := os.Getenv("USER_QUEUE_URL")
	log.Printf("User queue URL: %s", queueUrl)

	go func() { // Start processing in a goroutine
		for {
			log.Println("Attempting to receive messages from the user queue...")
			output, err := queueClient.ReceiveMessage(context.Background(), &sqs.ReceiveMessageInput{
				QueueUrl:            &queueUrl,
				MaxNumberOfMessages: 5,
				WaitTimeSeconds:     2,
			})
			if err != nil {
				log.Printf("Failed to receive user notification message: %v", err)
				time.Sleep(5 * time.Second)
				continue
			}

			log.Printf("Received %d messages from the user queue.", len(output.Messages))
			for _, msg := range output.Messages {
				log.Printf("Processing message with ID: %s", *msg.MessageId)
				var driverDTO dto.NotifyUserDTO
				err := json.Unmarshal([]byte(*msg.Body), &driverDTO)
				if err != nil {
					log.Printf("Invalid NotifyUser message: %v", err)
					continue
				}

				log.Printf("Enqueuing notification for user ID: %d", driverDTO.UserId)
				go n.notifyUserByFCM(driverDTO)

				log.Printf("Deleting message with ID: %s", *msg.MessageId)
				_, err = queueClient.DeleteMessage(context.Background(), &sqs.DeleteMessageInput{
					QueueUrl:      &queueUrl,
					ReceiptHandle: msg.ReceiptHandle,
				})
				if err != nil {
					log.Printf("Failed to delete NotifyUser message: %v", err)
				} else {
					log.Printf("Successfully deleted message with ID: %s", *msg.MessageId)
				}
			}
		}
	}()

	log.Println("NotifyUser process started successfully.")
	return nil
}

func (n *notifyUseCase) GetFcmToken(role string, accountId int) (string, error) {
	ctx := context.Background()
	var key string
	switch role {
	case "driver":
		key = fmt.Sprintf("fcm:driver:%d", accountId)
	case "customer":
		key = fmt.Sprintf("fcm:customer:%d", accountId)
	default:
		return "", fmt.Errorf("unsupported role: %s", role)
	}

	val, err := n.redis.Get(ctx, key).Result()
	if err != nil {
		log.Printf("Failed to retrieve FCM token: %v", err)
		return "", err
	}
	return val, nil
}

func (n *notifyUseCase) startNotifyDriverWorker(poolSize int) {

	poolSize = 5 // Default pool size if not set (hardcode ??!?!)

	for i := 0; i < poolSize; i++ {
		go func(id int) {
			for msg := range notifyChan {
				log.Printf("Worker-%d processing msg for driver ID %d", id, msg.DriverId)
				n.handleDriverNotification(msg)
			}
		}(i)
	}
}

func (n *notifyUseCase) handleDriverNotification(msg dto.NotifyDriverDTO) {
	log.Printf("Starting handleDriverNotification for driver ID %d", msg.DriverId)
	ctx := context.Background()

	log.Println("Initializing NotifyUseCase...")
	usecase := NewNotifyUseCase(n.redis, n.firebaseClient)

	log.Printf("Retrieving FCM token for driver ID %d...", msg.DriverId)
	token, err := usecase.GetFcmToken("driver", msg.DriverId)
	if err != nil {
		log.Printf("Failed to get FCM token for driver %d: %v", msg.DriverId, err)
		return
	}
	log.Printf("Successfully retrieved FCM token for driver ID %d", msg.DriverId)

	log.Println("Initializing Firebase Messaging client...")
	client, err := n.firebaseClient.Messaging(ctx)
	if err != nil {
		log.Printf("Failed to initialize FCM client: %v", err)
		return
	}
	log.Println("Firebase Messaging client initialized successfully.")

	// Compose message
	log.Printf("Composing notification message for driver ID %d...", msg.DriverId)
	message := &messaging.Message{
		Token: token,
		Data: map[string]string{
			"request_id":         msg.RequestId,
			"customer_full_name": msg.CustomerFullName,
			"customer_phone":     msg.CustomerPhone,
		},
	}

	log.Printf("Sending notification to driver ID %d for request ID %s...", msg.DriverId, msg.RequestId)
	_, err = client.Send(ctx, message)
	if err != nil {
		log.Printf("Failed to send FCM: %v", err)
	} else {
		log.Printf("Notification sent successfully to driver ID %d for request ID %s", msg.DriverId, msg.RequestId)
	}
}

func (n *notifyUseCase) notifyUserByFCM(notiyfUserDto dto.NotifyUserDTO) {
	log.Printf("Starting notifyUserByFCM for user ID %d and request ID %s...", notiyfUserDto.UserId, notiyfUserDto.RequestId)
	ctx := context.Background()

	log.Printf("Retrieving FCM token for user ID %d...", notiyfUserDto.UserId)
	token, err := n.GetFcmToken("customer", notiyfUserDto.UserId)
	if err != nil {
		log.Printf("Could not get FCM token for user %d: %v", notiyfUserDto.UserId, err)
		return
	}
	log.Printf("Successfully retrieved FCM token for user ID %d", notiyfUserDto.UserId)

	log.Println("Initializing Firebase Messaging client...")
	client, err := n.firebaseClient.Messaging(ctx)
	if err != nil {
		log.Printf("Failed to init Firebase client: %v", err)
		return
	}
	log.Println("Firebase Messaging client initialized successfully.")

	log.Printf("Composing notification message for user ID %d...", notiyfUserDto.UserId)
	message := &messaging.Message{
		Token: token,
		Data: map[string]string{
			"request_id": notiyfUserDto.RequestId,
			"full_name":  notiyfUserDto.Driver.FullName,
			"email":      notiyfUserDto.Driver.Email,
			"phone":      notiyfUserDto.Driver.Phone,
			"region_id":  strconv.Itoa(notiyfUserDto.Driver.RegionID),
			"car":        notiyfUserDto.Driver.Car,
		},
	}

	log.Printf("Sending notification to user ID %d for request ID %s...", notiyfUserDto.UserId, notiyfUserDto.RequestId)
	if _, err := client.Send(ctx, message); err != nil {
		log.Printf("FCM send to user %d failed: %v", notiyfUserDto.UserId, err)
	} else {
		log.Printf("Notification sent successfully to user ID %d for request ID %s", notiyfUserDto.UserId, notiyfUserDto.RequestId)
	}
	log.Printf("Finished notifyUserByFCM for user ID %d and request ID %s.", notiyfUserDto.UserId, notiyfUserDto.RequestId)
}
