package model

type RequestUpdateFcmTokenDTO struct {
	FcmToken string `json:"fcm_token" validate:"required"` // FCM token to update
	UserId   int    `json:"user_id" validate:"required"`   // ID of the user whose FCM token is being updated
	Role     string `json:"role" validate:"required"`      // Role of the user (e.g., "customer", "driver")
}
