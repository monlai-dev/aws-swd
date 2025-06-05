package customerfx

import (
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"gorm.io/gorm"
	"microservice_swd_demo/pkg/infra/cache"
	"microservice_swd_demo/pkg/infra/database"
	"microservice_swd_demo/service/customers/http"
	"microservice_swd_demo/service/customers/model/postgres"
	"microservice_swd_demo/service/customers/repository"
	"microservice_swd_demo/service/customers/usecase"
)

var Module = fx.Options(
	fx.Provide(
		provideCustomerDB,
		provideRedisDB,
		provideCustomerRepository,
		provideCustomerUseCase,
		provideCustomerHandler,
	),
	fx.Invoke(http.RegisterCustomerRoutes), // register routes
)

func provideRedisDB(lc fx.Lifecycle) *redis.Client {
	return cache.NewRedisClient(lc)
}

func provideCustomerDB(lc fx.Lifecycle) (*gorm.DB, error) {
	return database.ConnectWithSchema("customers", lc, &postgres.Customer{})
}

func provideCustomerRepository(dbConn *gorm.DB) repository.ICustomerRepository {
	return repository.NewCustomerRepository(dbConn)
}

func provideCustomerUseCase(customerRepo repository.ICustomerRepository, redis *redis.Client) usecase.ICustomerUseCase {
	return usecase.NewCustomerUseCase(customerRepo, redis)
}

func provideCustomerHandler(customerUseCase usecase.ICustomerUseCase) *http.AccountController {
	return http.NewAccountController(customerUseCase)
}
