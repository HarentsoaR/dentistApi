package utils

import (
	"errors"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret = []byte(os.Getenv("JWT_SECRET"))

type Claims struct {
	UserID string `json:"userId"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateJWT creates a new JWT token for a given user.
func GenerateJWT(userID, role string) (string, error) {
	if len(jwtSecret) == 0 {
		log.Println("CRITICAL: JWT_SECRET is not configured. Cannot generate token.")
		return "", errors.New("JWT_SECRET is not configured")
	}
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ValidateJWT validates a given token string.
func ValidateJWT(tokenStr string) (*Claims, error) {
	if len(jwtSecret) == 0 {
		log.Println("CRITICAL: JWT_SECRET is not configured. Cannot validate token.")
		return nil, errors.New("JWT_SECRET is not configured")
	}
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return nil, err
	}

	return claims, nil
}
