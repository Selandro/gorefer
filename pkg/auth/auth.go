package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorefer.go/pkg/storage"
)

// Хэширование пароля
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// Проверка пароля
func CheckPasswordHash(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// Обработчик для регистрации пользователя
func RegisterHandler(db storage.DBInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var user storage.User
		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		hashedPassword, err := HashPassword(user.Password)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		user.Password = hashedPassword

		// Создаем контекст с таймаутом для взаимодействия с базой данных
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		if _, err := db.CreateUser(ctx, user); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}

// Обработчик для аутентификации пользователя
func LoginHandler(db storage.DBInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var user storage.User
		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Создаем контекст с таймаутом для взаимодействия с базой данных
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		existingUser, err := db.GetUserByEmail(ctx, user.Email)
		if err != nil {
			http.Error(w, "Неверный логин или пароль", http.StatusUnauthorized)
			return
		}

		if err := CheckPasswordHash(user.Password, existingUser.Password); err != nil {
			http.Error(w, "Неверный логин или пароль", http.StatusUnauthorized)
			return
		}

		token, err := GenerateToken(existingUser.ID, existingUser.Username)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Возвращаем токен в теле ответа
		response := map[string]string{
			"token": token,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}
