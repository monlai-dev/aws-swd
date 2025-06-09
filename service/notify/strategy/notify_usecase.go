// File: strategy/token_strategy.go
package strategy

import (
	"context"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
	"microservice_swd_demo/service/notify/model"
)

type TokenStrategy interface {
	StoreToken(ctx context.Context, rdb *redis.Client, dto model.RequestUpdateFcmTokenDTO) error
}

type CustomerTokenStrategy struct{}

type DriverTokenStrategy struct{}

func (c CustomerTokenStrategy) StoreToken(ctx context.Context, rdb *redis.Client, dto model.RequestUpdateFcmTokenDTO) error {
	key := fmt.Sprintf("fcm:customer:%d", dto.UserId)
	return rdb.Set(ctx, key, dto.FcmToken, 0).Err()
}

func (d DriverTokenStrategy) StoreToken(ctx context.Context, rdb *redis.Client, dto model.RequestUpdateFcmTokenDTO) error {
	key := fmt.Sprintf("fcm:driver:%d", dto.UserId)
	return rdb.Set(ctx, key, dto.FcmToken, 0).Err()
}

func GetTokenStrategy(role string) (TokenStrategy, error) {
	switch role {
	case "customer":
		return CustomerTokenStrategy{}, nil
	case "driver":
		return DriverTokenStrategy{}, nil
	default:
		return nil, errors.New("unsupported role: " + role)
	}
}
