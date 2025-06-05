package repository

import (
	"gorm.io/gorm"
	"microservice_swd_demo/service/drivers/model/postgres"
)

type IDriverRepository interface {
	GetByID(id int) (postgres.Driver, error)
}

type driverRepository struct {
	dbConn *gorm.DB
}

func NewDriverRepository(dbConn *gorm.DB) IDriverRepository {
	return &driverRepository{
		dbConn: dbConn,
	}
}

func (d driverRepository) GetByID(id int) (postgres.Driver, error) {

	var driver postgres.Driver
	if err := d.dbConn.First(&driver, id).Error; err != nil {
		return postgres.Driver{}, err
	}

	return driver, nil
}
