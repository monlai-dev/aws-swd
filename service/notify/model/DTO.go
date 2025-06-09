package model

type NotifyDriverDTO struct {
	DriverId         int    `json:"driver_id"`
	RequestId        string `json:"request_id"`
	CustomerFullName string `json:"customer_full_name"`
	CustomerPhone    string `json:"customer_phone"`
}

type NotifyUserDTO struct {
	RequestId string            `json:"request_id"`
	UserId    int               `json:"user_id"`
	Driver    DriverResponseDTO `json:"driver"`
}

type DriverResponseDTO struct {
	FullName string `json:"full_name"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	RegionID int    `json:"region_id"`
	Car      string `json:"car"`
}
