package postgres

import "gorm.io/gorm"

type Driver struct {
	gorm.Model
	Name     string `gorm:"type:varchar(100);not null" json:"name"`
	Phone    string `gorm:"type:varchar(20);unique;not null" json:"phone"`
	Email    string `gorm:"type:varchar(100);unique" json:"email"`
	RegionID int    `gorm:"type:int;not null" json:"regionId"`
	Car      string `gorm:"type:varchar(100);not null" json:"car"`
}
