// File: strategy/token_strategy.go
package strategy

import (
	"context"
	"errors"
	"fmt"
	"microservice_swd_demo/service/notify/model/dto"

	"github.com/redis/go-redis/v9"
)

type TokenStrategy interface {
	StoreToken(ctx context.Context, rdb *redis.Client, dto dto.RequestUpdateFcmTokenDTO) error
}

type CustomerTokenStrategy struct{}

type DriverTokenStrategy struct{}

func (c CustomerTokenStrategy) StoreToken(ctx context.Context, rdb *redis.Client, dto dto.RequestUpdateFcmTokenDTO) error {
	key := fmt.Sprintf("fcm:customer:%d", dto.UserId)
	return rdb.Set(ctx, key, dto.FcmToken, 0).Err()
}

func (d DriverTokenStrategy) StoreToken(ctx context.Context, rdb *redis.Client, dto dto.RequestUpdateFcmTokenDTO) error {
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
