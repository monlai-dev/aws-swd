package request

type CreateAccountDTO struct {
	Email    string `gorm:"type:varchar(100);not null" json:"email" validate:"required,email"`
	Password string `gorm:"type:varchar(64);not null" json:"password" validate:"required,min=8,max=64"`
	FullName string `gorm:"type:varchar(64);not null" json:"full_name" validate:"required,min=2,max=64"`
	Phone    string `gorm:"type:varchar(15);unique;not null" json:"phone" validate:"required,min=10,max=15"`
	RegionID int    `gorm:"type:int;not null" json:"region_id" validate:"required"`
}

type Login struct {
	Email    string `gorm:"type:varchar(100);not null" json:"email" validate:"required,email"`
	Password string `gorm:"type:varchar(64);not null" json:"password" validate:"required,min=8,max=64"`
}

type GetAccountInfoWithId struct {
	CustomerId int `json:"cusomer_id" validate:"required"`
}

type RideRequestDTO struct {
	UserId   int    `json:"user_id" validate:"required"` // ID of the user making the request
	RegionId string `json:"region" validate:"required"`  // Region where the ride is requested
}
