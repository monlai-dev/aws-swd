package helper

import (
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func ComparePasswords(hashedPassword string, plainPassword string) error {

	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainPassword))

}
