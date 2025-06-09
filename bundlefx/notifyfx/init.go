package notifyfx

import (
	firebase "firebase.google.com/go/v4"
	"microservice_swd_demo/pkg/infra/firebaseClient"
	"microservice_swd_demo/service/notify/http"

	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"microservice_swd_demo/pkg/infra/cache"
	"microservice_swd_demo/service/notify/usecase"
)

var Module = fx.Options(
	fx.Provide(
		provideRedisDB,
		provideFirebaseClient,
		provideNotifyUseCase,
	),
	fx.Invoke(usecase.INotifyUseCase.NotifyDriver),
	fx.Invoke(usecase.INotifyUseCase.NotifyUser),
	fx.Invoke(provideNotifyController),
)

func provideRedisDB(lc fx.Lifecycle) *redis.Client {
	return cache.NewRedisClient(lc)
}

func provideNotifyUseCase(redis *redis.Client, firebaseClient *firebase.App) usecase.INotifyUseCase {
	return usecase.NewNotifyUseCase(redis, firebaseClient)
}

func provideFirebaseClient() *firebase.App {
	return firebaseClient.ProvideFirebaseApp()
}

func provideNotifyController(usecase usecase.INotifyUseCase) *http.NotifyController {
	return http.NewNotifyController(usecase)
}
