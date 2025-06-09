package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"microservice_swd_demo/service/notify/strategy"
	"os"
	"strconv"
	"time"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/redis/go-redis/v9"
	"microservice_swd_demo/service/notify/model"
)

var (
	queueClient *sqs.Client
	notifyChan  = make(chan model.NotifyDriverDTO, 100)
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
	RegisterFcmToken(dto model.RequestUpdateFcmTokenDTO) error
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
	queueUrl := os.Getenv("DRIVER_QUEUE_URL")

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
			var dto model.NotifyDriverDTO
			err := json.Unmarshal([]byte(*msg.Body), &dto)
			if err != nil {
				log.Printf("Failed to parse message body: %v", err)
				continue
			}

			notifyChan <- dto // Push to worker channel

			// Delete message after enqueue
			_, err = queueClient.DeleteMessage(context.Background(), &sqs.DeleteMessageInput{
				QueueUrl:      &queueUrl,
				ReceiptHandle: msg.ReceiptHandle,
			})
			if err != nil {
				log.Printf("Failed to delete message: %v", err)
			}
		}
	}
}

func (n *notifyUseCase) RegisterFcmToken(dto model.RequestUpdateFcmTokenDTO) error {
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
	queueUrl := os.Getenv("USER_QUEUE_URL")
	for {
		output, err := queueClient.ReceiveMessage(context.Background(), &sqs.ReceiveMessageInput{
			QueueUrl:            &queueUrl,
			MaxNumberOfMessages: 5,
			WaitTimeSeconds:     2,
		})
		if err != nil {
			log.Printf("Failed to receive user noti msg: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		for _, msg := range output.Messages {
			var dto model.NotifyUserDTO
			err := json.Unmarshal([]byte(*msg.Body), &dto)
			if err != nil {
				log.Printf("Invalid NotifyUser message: %v", err)
				continue
			}

			go n.notifyUserByFCM(dto)

			_, err = queueClient.DeleteMessage(context.Background(), &sqs.DeleteMessageInput{
				QueueUrl:      &queueUrl,
				ReceiptHandle: msg.ReceiptHandle,
			})
			if err != nil {
				log.Printf("Failed to delete NotifyUser message: %v", err)
			}
		}
	}
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

func (n *notifyUseCase) handleDriverNotification(msg model.NotifyDriverDTO) {
	ctx := context.Background()

	usecase := NewNotifyUseCase(n.redis, n.firebaseClient)

	token, err := usecase.GetFcmToken("driver", msg.DriverId)
	if err != nil {
		log.Printf("Failed to get FCM token for driver %d: %v", msg.DriverId, err)
		return
	}

	client, err := n.firebaseClient.Messaging(ctx)
	if err != nil {
		log.Printf("Failed to initialize FCM client: %v", err)
		return
	}

	// Compose message
	message := &messaging.Message{
		Token: token,
		Data: map[string]string{
			"request_id":         msg.RequestId,
			"customer_full_name": msg.CustomerFullName,
			"customer_phone":     msg.CustomerPhone,
		},
	}

	_, err = client.Send(ctx, message)
	if err != nil {
		log.Printf("Failed to send FCM: %v", err)
	} else {
		log.Printf("Notification sent to driver for request ID %s", msg.RequestId)
	}
}

func (n *notifyUseCase) notifyUserByFCM(dto model.NotifyUserDTO) {
	ctx := context.Background()
	token, err := n.GetFcmToken("customer", dto.UserId)
	if err != nil {
		log.Printf("Could not get FCM token for user %d: %v", dto.UserId, err)
		return
	}

	client, err := n.firebaseClient.Messaging(ctx)
	if err != nil {
		log.Printf("Failed to init Firebase client: %v", err)
		return
	}

	message := &messaging.Message{
		Token: token,
		Data: map[string]string{
			"request_id": dto.RequestId,
			"full_name":  dto.Driver.FullName,
			"email":      dto.Driver.Email,
			"phone":      dto.Driver.Phone,
			"region_id":  strconv.Itoa(dto.Driver.RegionID),
			"car":        dto.Driver.Car,
		},
	}

	if _, err := client.Send(ctx, message); err != nil {
		log.Printf("FCM send to user %d failed: %v", dto.UserId, err)
	} else {
		log.Printf("Push sent to user %d for ride %s", dto.UserId, dto.RequestId)
	}
}
