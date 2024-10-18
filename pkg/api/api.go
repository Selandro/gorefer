package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"gorefer.go/pkg/auth" // Импортируем пакет auth
	"gorefer.go/pkg/storage"
)

// API структура.
type API struct {
	db storage.DBInterface
	r  *chi.Mux
}

// Конструктор API.
func New(db storage.DBInterface) *API {
	a := API{db: db, r: chi.NewRouter()}
	a.endpoints()
	return &a
}

// Router возвращает маршрутизатор для использования
// в качестве аргумента HTTP-сервера.
func (api *API) Router() *chi.Mux {
	return api.r
}

type contextKey string

const (
	UserKey contextKey = "username"
)

// Middleware для проверки токена
func TokenAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString := r.Header.Get("Authorization")
		if tokenString == "" || len(tokenString) < len("Bearer ") {
			http.Error(w, "Токен не предоставлен", http.StatusUnauthorized)
			return
		}

		// Удаляем "Bearer " из токена
		tokenString = tokenString[len("Bearer "):]

		username, err := auth.ValidateToken(tokenString)
		if err != nil {
			http.Error(w, "Недействительный токен", http.StatusUnauthorized)
			fmt.Println("Ошибка при проверке токена:", err)
			return
		}

		// Добавляем имя пользователя в контекст запроса
		ctx := context.WithValue(r.Context(), UserKey, username)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

// Регистрация методов API в маршрутизаторе запросов.
func (api *API) endpoints() {
	// Middleware для логирования запросов
	api.r.Use(middleware.Logger)

	// Регистрация пользователей (без токена)
	api.r.Post("/register", api.registerUser)
	api.r.Post("/register-with-referral", api.registerWithReferralCode)
	// Аутентификация пользователей (без токена)
	api.r.Post("/login", api.loginUser)

	// Группа защищенных маршрутов
	api.r.Route("/protected", func(r chi.Router) {
		r.Use(TokenAuthMiddleware) // Применение middleware для защищённых маршрутов

		r.Post("/referral-code", api.createReferralCode)
		r.Delete("/referral-code", api.deleteReferralCode)
		r.Get("/referral-code/{email}", api.getReferralCodeByEmail)
		r.Get("/referrals/{referrerID}", api.getReferralsByReferrerID)
	})
}

// Обработчик для регистрации пользователя
func (api *API) registerUser(w http.ResponseWriter, r *http.Request) {
	var user storage.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	hashedPassword, err := auth.HashPassword(user.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	user.Password = hashedPassword

	if _, err := api.db.CreateUser(user); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// Обработчик для аутентификации пользователя
func (api *API) loginUser(w http.ResponseWriter, r *http.Request) {
	var user storage.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	existingUser, err := api.db.GetUserByEmail(user.Email)
	if err != nil {
		http.Error(w, "Неверный логин или пароль", http.StatusUnauthorized)
		return
	}

	if err := auth.CheckPasswordHash(user.Password, existingUser.Password); err != nil {
		http.Error(w, "Неверный логин или пароль", http.StatusUnauthorized)
		return
	}

	token, err := auth.GenerateToken(existingUser.ID, existingUser.Username)
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

// Обработчик для создания реферального кода
func (api *API) createReferralCode(w http.ResponseWriter, r *http.Request) {
	var request struct {
		UserID    int    `json:"user_id"`
		Code      string `json:"code"`
		ExpiresAt int64  `json:"expires_at"` // Временная метка истечения
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := api.db.CreateReferralCode(request.UserID, request.Code, request.ExpiresAt); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// Обработчик для удаления реферального кода
func (api *API) deleteReferralCode(w http.ResponseWriter, r *http.Request) {
	var request struct {
		UserID int `json:"user_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := api.db.DeleteReferralCode(request.UserID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Обработчик для получения реферального кода по email
func (api *API) getReferralCodeByEmail(w http.ResponseWriter, r *http.Request) {
	email := chi.URLParam(r, "email")

	referralCode, err := api.db.GetReferralCodeByEmail(email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(referralCode)
}

// Обработчик для регистрации по реферальному коду
func (api *API) registerWithReferralCode(w http.ResponseWriter, r *http.Request) {
	var request struct {
		ReferralCode string       `json:"referral_code"`
		User         storage.User `json:"user"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := api.db.RegisterWithReferralCode(request.ReferralCode, request.User); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// Обработчик для получения рефералов по ID реферера
func (api *API) getReferralsByReferrerID(w http.ResponseWriter, r *http.Request) {
	referrerID := chi.URLParam(r, "referrerID")

	// Преобразование referrerID в int
	id, err := strconv.Atoi(referrerID)
	if err != nil {
		http.Error(w, "Invalid referrer ID", http.StatusBadRequest)
		return
	}

	referrals, err := api.db.GetReferralsByReferrerID(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(referrals)
}
