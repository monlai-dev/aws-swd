package repository

import (
	"errors"
	"gorm.io/gorm"
	"log"
	"microservice_swd_demo/service/customers/model/postgres"
)

type ICustomerRepository interface {
	GetByID(id int) (postgres.Customer, error)
	GetByUsernameAndPassword(username string) (postgres.Customer, error)
	CreateAccount(customer postgres.Customer) error
}

type customerRepository struct {
	dbConn *gorm.DB
}

func NewCustomerRepository(dbConn *gorm.DB) ICustomerRepository {
	return &customerRepository{
		dbConn: dbConn,
	}
}

func (c customerRepository) GetByID(id int) (postgres.Customer, error) {
	var customer postgres.Customer
	if err := c.dbConn.First(&customer, id).Error; err != nil {
		return postgres.Customer{}, err
	}
	return customer, nil
}

func (c customerRepository) GetByUsernameAndPassword(username string) (postgres.Customer, error) {

	var customer postgres.Customer
	if err := c.dbConn.Where("email = ?", username).First(&customer).Error; err != nil {
		return postgres.Customer{}, err
	}
	return customer, nil
}

func (c customerRepository) CreateAccount(customer postgres.Customer) error {

	if err := c.dbConn.Create(&customer).Error; err != nil {
		log.Printf("failed to create customer account: %v", err)
		return errors.New("failed to create customer account: ")
	}
	return nil
}
