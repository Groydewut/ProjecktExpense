package auth

import (
	"errors"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func ValidateToken(tokenStr string) (int, error) {
	// Достаем тот же самый секрет из .env
	secretStr := os.Getenv("JWT_SECRET")
	if strings.TrimSpace(secretStr) == "" {
		secretStr = "default_secret_key=)"
	}

	jwtSecret := []byte(secretStr)
	// Парсим токен и проверяем его подпись
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		// Проверяем, что алгоритм подписи — именно HMAC (HS256)
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("не верный код подписи")
		}
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return 0, errors.New("не валидный токен")
	}
	// Если токен валиден, достаем из него Claims (наш user_id)
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, errors.New("не удалось прочитать claims")
	}
	// Превращаем float64 (в JSON все числа парсятся как float64) в int
	userID, ok := claims["user_id"].(float64)
	if !ok {
		return 0, errors.New("user_id отсутствует в токене")
	}
	return int(userID), nil
}

func GenerateToken(userID int) (string, error) {

	secretStr := os.Getenv("JWT_SECRET")
	if strings.TrimSpace(secretStr) == "" {
		secretStr = "default_secret_key=)"
	}

	jwtSecret := []byte(secretStr)

	// 1. Создаем Claims (заявления) — то, что будет зашито внутри токена
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 24).Unix(), //токен сгорит через 24 часа
	}
	// 2. Создаем сам токен с указанием алгоритма и наших данных
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}
