package request

type GetRequestById struct {
	RequestId int `json:"request_id" validate:"required"` // ID of the request to retrieve
}

type AcceptOrderDTO struct {
	RequestId string `json:"request_id" validate:"required"` // ID of the request to accept
	DriverId  int    `json:"driver_id" validate:"required"`  // ID of the driver accepting the request
}

type OnlineRequestDTO struct {
	DriverId int `json:"driver_id" validate:"required"` // ID of the customer making the request
	RegionId int `json:"region_id" validate:"required"` // ID of the region where the request is made
}
