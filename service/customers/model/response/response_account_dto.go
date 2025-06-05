package response

type AccountResponseDTO struct {
	CustomerID int    `json:"customer_id"`
	Email      string `json:"email"`
	Phone      string `json:"phone"`
	FullName   string `json:"full_name"`
	RegionID   int    `json:"region_id"`
}

type LoginResponseDTO struct {
	AccessToken string `json:"access_token"`
	UserId      int    `json:"user_id"`
}
