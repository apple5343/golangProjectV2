package jwt

import (
	"fmt"
	"time"

	"github.com/apple5343/golangProjectV2/internal/domain/models"

	"github.com/golang-jwt/jwt/v5"
)

func NewToken(user models.User, secret string, duration time.Duration) (string, error) {
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"isAdmin": user.IsAdmin,
		"id":      user.ID,
		"nbf":     now.Unix(),
		"exp":     now.Add(duration).Unix(),
		"iat":     now.Unix(),
	})

	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func TokenValues(token, secret string) (jwt.MapClaims, error) {
	tokenFromString, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Неправильный токен")
		}

		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := tokenFromString.Claims.(jwt.MapClaims); ok {
		return claims, nil
	} else {
		return nil, fmt.Errorf("Ошибка чтения")
	}
}
