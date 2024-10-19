package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"gorefer.go/pkg/api/middlware"
	"gorefer.go/pkg/auth"
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

// Регистрация методов API в маршрутизаторе запросов.
func (api *API) endpoints() {
	api.r.Use(middleware.Logger)

	api.r.Post("/register", api.registerUser)
	api.r.Post("/register-with-referral", api.registerWithReferralCode)
	api.r.Post("/login", api.loginUser)

	api.r.Route("/protected", func(r chi.Router) {
		r.Use(middlware.TokenAuthMiddleware)
		r.Post("/referral-code", api.createReferralCode)
		r.Delete("/referral-code", api.deleteReferralCode)
		r.Get("/referral-code/{email}", api.getReferralCodeByEmail)
		r.Get("/referrals/{referrerID}", api.getReferralsByReferrerID)
	})
}

// Функция для обработки ошибок
func (api *API) writeError(w http.ResponseWriter, message string, code int) {
	http.Error(w, message, code)
}

// Функция для создания контекста с таймаутом
func (api *API) withTimeout(ctx context.Context, duration time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, duration)
}

// Обработчик для регистрации пользователя
func (api *API) registerUser(w http.ResponseWriter, r *http.Request) {
	var user storage.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		api.writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := api.withTimeout(r.Context(), 5*time.Second)
	defer cancel()

	resultChan := make(chan error)
	go func() {
		hashedPassword, err := auth.HashPassword(user.Password)
		if err != nil {
			resultChan <- err
			return
		}
		user.Password = hashedPassword
		_, err = api.db.CreateUser(ctx, user)
		resultChan <- err
	}()

	if err := <-resultChan; err != nil {
		api.writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// Обработчик для аутентификации пользователя
func (api *API) loginUser(w http.ResponseWriter, r *http.Request) {
	var user storage.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		api.writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := api.withTimeout(r.Context(), 5*time.Second)
	defer cancel()

	resultChan := make(chan storage.User)
	errorChan := make(chan error)

	go func() {
		existingUser, err := api.db.GetUserByEmail(ctx, user.Email)
		if err != nil {
			errorChan <- err
			return
		}
		resultChan <- existingUser
	}()

	select {
	case existingUser := <-resultChan:
		if err := auth.CheckPasswordHash(user.Password, existingUser.Password); err != nil {
			api.writeError(w, "Неверный логин или пароль", http.StatusUnauthorized)
			return
		}

		token, err := auth.GenerateToken(existingUser.ID, existingUser.Username)
		if err != nil {
			api.writeError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		response := map[string]string{"token": token}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)

	case err := <-errorChan:
		api.writeError(w, err.Error(), http.StatusUnauthorized)
		return
	}
}

// Обработчик для создания реферального кода
func (api *API) createReferralCode(w http.ResponseWriter, r *http.Request) {
	var request struct {
		UserID    int    `json:"user_id"`
		Code      string `json:"code"`
		ExpiresAt int64  `json:"expires_at"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		api.writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := api.withTimeout(r.Context(), 5*time.Second)
	defer cancel()

	resultChan := make(chan error)
	go func() {
		err := api.db.CreateReferralCode(ctx, request.UserID, request.Code, request.ExpiresAt)
		resultChan <- err
	}()

	if err := <-resultChan; err != nil {
		api.writeError(w, err.Error(), http.StatusInternalServerError)
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
		api.writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := api.withTimeout(r.Context(), 5*time.Second)
	defer cancel()

	resultChan := make(chan error)
	go func() {
		err := api.db.DeleteReferralCode(ctx, request.UserID)
		resultChan <- err
	}()

	if err := <-resultChan; err != nil {
		api.writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Обработчик для получения реферального кода по email
func (api *API) getReferralCodeByEmail(w http.ResponseWriter, r *http.Request) {
	email := chi.URLParam(r, "email")

	ctx, cancel := api.withTimeout(r.Context(), 5*time.Second)
	defer cancel()

	resultChan := make(chan *storage.ReferralCode)
	errorChan := make(chan error)

	go func() {
		referralCode, err := api.db.GetReferralCodeByEmail(ctx, email)
		if err != nil {
			errorChan <- err
			return
		}
		resultChan <- &referralCode
	}()

	select {
	case referralCode := <-resultChan:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(referralCode)

	case err := <-errorChan:
		api.writeError(w, err.Error(), http.StatusNotFound)
		return
	}
}

// Обработчик для регистрации по реферальному коду
func (api *API) registerWithReferralCode(w http.ResponseWriter, r *http.Request) {
	var request struct {
		ReferralCode string       `json:"referral_code"`
		User         storage.User `json:"user"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		api.writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := api.withTimeout(r.Context(), 5*time.Second)
	defer cancel()

	resultChan := make(chan error)
	go func() {
		err := api.db.RegisterWithReferralCode(ctx, request.ReferralCode, request.User)
		resultChan <- err
	}()

	if err := <-resultChan; err != nil {
		api.writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// Обработчик для получения рефералов по ID реферера
func (api *API) getReferralsByReferrerID(w http.ResponseWriter, r *http.Request) {
	referrerID := chi.URLParam(r, "referrerID")

	id, err := strconv.Atoi(referrerID)
	if err != nil {
		api.writeError(w, "Invalid referrer ID", http.StatusBadRequest)
		return
	}

	ctx, cancel := api.withTimeout(r.Context(), 5*time.Second)
	defer cancel()

	resultChan := make(chan []storage.User)
	errorChan := make(chan error)

	go func() {
		referrals, err := api.db.GetReferralsByReferrerID(ctx, id)
		if err != nil {
			errorChan <- err
			return
		}
		resultChan <- referrals
	}()

	select {
	case referrals := <-resultChan:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(referrals)

	case err := <-errorChan:
		api.writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
