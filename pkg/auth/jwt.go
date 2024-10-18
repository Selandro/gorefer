package auth

import (
	"errors"
	"os"
	"time"

	"github.com/dgrijalva/jwt-go"
)

// JWTSecret - секретный ключ для подписи токенов (загружаем из переменной окружения)
var JWTSecret = []byte(os.Getenv("JWT_SECRET"))

// CustomClaims включает стандартные и дополнительные поля
type CustomClaims struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	jwt.StandardClaims
}

// Создание JWT токена с кастомными утверждениями
func GenerateToken(userID int, username string) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)

	claims := &CustomClaims{
		UserID:   userID,
		Username: username,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
			IssuedAt:  time.Now().Unix(),
			Subject:   username,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JWTSecret)
}

// Проверка JWT токена с кастомными утверждениями
func ValidateToken(tokenString string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("недопустимый метод подписи")
		}
		return JWTSecret, nil
	})

	if err != nil {
		if err == jwt.ErrSignatureInvalid {
			return "", errors.New("недействительная подпись токена")
		}
		return "", errors.New("ошибка разбора токена: " + err.Error())
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return "", errors.New("недействительный токен")
	}

	// Проверяем истечение токена
	if claims.ExpiresAt < time.Now().Unix() {
		return "", errors.New("токен истек")
	}

	return claims.Username, nil
}
