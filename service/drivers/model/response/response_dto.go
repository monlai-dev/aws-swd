package response

type DriverResponseDTO struct {
	FullName string `json:"full_name"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	RegionID int    `json:"region_id"`
	Car      string `json:"car"`
}
