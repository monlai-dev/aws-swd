package postgres

import "gorm.io/gorm"

type Customer struct {
	gorm.Model
	Name     string `gorm:"type:varchar(100);not null" json:"name"`
	Password string `gorm:"type:varchar(100);not null" json:"password"`
	Phone    string `gorm:"type:varchar(20);unique;not null" json:"phone"`
	Email    string `gorm:"type:varchar(100);unique" json:"email"`
	RegionID int    `gorm:"type:int;not null" json:"regionId"`
}
