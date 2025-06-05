package driverfx

import (
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"gorm.io/gorm"
	"microservice_swd_demo/pkg/infra/cache"
	"microservice_swd_demo/pkg/infra/database"
	"microservice_swd_demo/service/drivers/http"
	"microservice_swd_demo/service/drivers/model/postgres"
	"microservice_swd_demo/service/drivers/repository"
	"microservice_swd_demo/service/drivers/usecase"
)

var Module = fx.Options(
	fx.Provide(provideDriverDb,
		provideRedisDB,
		provideDriverRepo,
		provideDriverUseCase,
		provideDriverHandler),
	fx.Invoke(http.RegisterRoutes),
	fx.Invoke(usecase.IDriverUseCase.MatchingOrder))

func provideDriverDb(lc fx.Lifecycle) (*gorm.DB, error) {
	return database.ConnectWithSchema("drivers", lc, &postgres.Driver{})
}

func provideRedisDB(lc fx.Lifecycle) *redis.Client {
	return cache.NewRedisClient(lc)
}

func provideDriverRepo(db *gorm.DB) repository.IDriverRepository {
	return repository.NewDriverRepository(db)
}

func provideDriverUseCase(repo repository.IDriverRepository, redis *redis.Client) usecase.IDriverUseCase {
	return usecase.NewDriverUseCase(repo, redis)
}

func provideDriverHandler(useCase usecase.IDriverUseCase) *http.DriversHandler {
	return http.NewDriversHandler(useCase)
}
