package utils

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/krasvl/market/internal/storage"
)

func GenerateToken(user storage.User, secret string) (string, error) {
	claims := jwt.MapClaims{
		"userID": user.ID,
		"exp":    time.Now().Add(time.Hour * 72).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func ParseToken(tokenStr, secret string) (int, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})

	if err != nil {
		return 0, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userID, ok := claims["userID"].(float64)
		if !ok {
			return 0, errors.New("invalid token claims")
		}
		return int(userID), nil
	}

	return 0, errors.New("invalid token")
}
