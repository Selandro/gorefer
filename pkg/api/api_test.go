package api_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"gorefer.go/pkg/api"
	"gorefer.go/pkg/auth"
	"gorefer.go/pkg/storage"
)

func TestAPI_RegisterUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := storage.NewMockDBInterface(ctrl)
	apiHandler := api.New(mockDB)

	tests := []struct {
		name         string
		input        storage.User
		expectedCode int
		mockSetup    func()
	}{
		{
			name: "Successful registration",
			input: storage.User{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "password123",
			},
			expectedCode: http.StatusCreated,
			mockSetup: func() {
				mockDB.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Return(1, nil) // возврат успешного результата
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mockSetup != nil {
				tt.mockSetup() // установка ожиданий
			}

			body, _ := json.Marshal(tt.input)                                       // сериализация входных данных в JSON
			req, err := http.NewRequest("POST", "/register", bytes.NewBuffer(body)) // создание HTTP-запроса
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()                               // создаем новый тестовый респондер
			handler := http.HandlerFunc(apiHandler.Router().ServeHTTP) // получаем обработчик
			handler.ServeHTTP(rr, req)                                 // выполняем запрос

			if status := rr.Code; status != tt.expectedCode {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.expectedCode)
			}
		})
	}
}

func TestAPI_LoginUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := storage.NewMockDBInterface(ctrl)
	apiHandler := api.New(mockDB)

	tests := []struct {
		name         string
		input        storage.User
		expectedCode int
		mockSetup    func()
	}{
		{
			name: "Successful login",
			input: storage.User{
				Email:    "test@example.com",
				Password: "password123",
			},
			expectedCode: http.StatusOK,
			mockSetup: func() {
				// Hash the password for the mock return
				hashedPassword, _ := auth.HashPassword("password123")
				mockDB.EXPECT().
					GetUserByEmail(gomock.Any(), "test@example.com").
					Return(storage.User{
						ID:       1,
						Username: "testuser",
						Password: hashedPassword, // Hashed password
					}, nil)
			},
		},
		{
			name: "User not found",
			input: storage.User{
				Email:    "notfound@example.com",
				Password: "password123",
			},
			expectedCode: http.StatusUnauthorized,
			mockSetup: func() {
				mockDB.EXPECT().
					GetUserByEmail(gomock.Any(), "notfound@example.com").
					Return(storage.User{}, errors.New("user not found"))
			},
		},
		{
			name: "Incorrect password",
			input: storage.User{
				Email:    "test@example.com",
				Password: "wrongpassword",
			},
			expectedCode: http.StatusUnauthorized,
			mockSetup: func() {
				// Hash the correct password for the mock return
				hashedPassword, _ := auth.HashPassword("password123")
				mockDB.EXPECT().
					GetUserByEmail(gomock.Any(), "test@example.com").
					Return(storage.User{
						ID:       1,
						Username: "testuser",
						Password: hashedPassword, // Hashed password
					}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mockSetup != nil {
				tt.mockSetup() // Setup the mock expectations
			}

			body, _ := json.Marshal(tt.input)                                    // Serialize input to JSON
			req, err := http.NewRequest("POST", "/login", bytes.NewBuffer(body)) // Create HTTP request
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()                               // Create a new ResponseRecorder
			handler := http.HandlerFunc(apiHandler.Router().ServeHTTP) // Get the handler
			handler.ServeHTTP(rr, req)                                 // Perform the request

			if status := rr.Code; status != tt.expectedCode {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.expectedCode)
			}

			if rr.Body.String() == "" && http.StatusOK == 200 {
				t.Error("Expected a non-empty response body for successful login")
			}
		})
	}
}

func TestAPI_RegisterWithReferralCode(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := storage.NewMockDBInterface(ctrl)
	apiHandler := api.New(mockDB)

	tests := []struct {
		name         string
		input        storage.User
		referralCode string
		expectedCode int
		mockSetup    func()
	}{
		{
			name: "Successful registration without referral code",
			input: storage.User{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "password123",
			},
			referralCode: "",
			expectedCode: http.StatusCreated,
			mockSetup: func() {
				mockDB.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Return(1, nil) // успешная регистрация
			},
		},
		{
			name: "Successful registration with referral code",
			input: storage.User{
				Username: "testuser2",
				Email:    "test2@example.com",
				Password: "password123",
			},
			referralCode: "REF123",
			expectedCode: http.StatusCreated,
			mockSetup: func() {
				mockDB.EXPECT().
					RegisterWithReferralCode(gomock.Any(), "REF123", gomock.Any()).
					Return(nil) // успешное применение реферального кода
			},
		},
		{
			name: "Failed registration with referral code",
			input: storage.User{
				Username: "testuser3",
				Email:    "test3@example.com",
				Password: "password123",
			},
			referralCode: "REF123",
			expectedCode: http.StatusInternalServerError,
			mockSetup: func() {
				mockDB.EXPECT().
					RegisterWithReferralCode(gomock.Any(), "REF123", gomock.Any()).
					Return(errors.New("some database error")) // имитируем ошибку
			},
		},
		{
			name: "Failed to decode request payload",
			input: storage.User{
				Username: "testuser4",
				Email:    "test4@example.com",
				Password: "password123",
			},
			referralCode: "", // неважно
			expectedCode: http.StatusBadRequest,
			mockSetup:    func() {}, // Не требуется
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mockSetup != nil {
				tt.mockSetup() // Установка ожиданий
			}

			var body *bytes.Buffer
			if tt.name == "Failed to decode request payload" {
				// Пустой или некорректный JSON
				body = bytes.NewBuffer([]byte("invalid json"))
			} else {
				requestBody := struct {
					ReferralCode string       `json:"referral_code,omitempty"`
					User         storage.User `json:"user"`
				}{
					ReferralCode: tt.referralCode,
					User:         tt.input,
				}
				bodyBytes, _ := json.Marshal(requestBody)
				body = bytes.NewBuffer(bodyBytes)
			}

			req, err := http.NewRequest("POST", "/register-with-referral", body)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()                               // создаем новый тестовый респондер
			handler := http.HandlerFunc(apiHandler.Router().ServeHTTP) // получаем обработчик
			handler.ServeHTTP(rr, req)                                 // выполняем запрос

		})
	}
}
