package storage

import (
	"context"
	"testing"
	"time"

	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestDB_CreateUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := NewMockDBInterface(ctrl)

	tests := []struct {
		name    string
		user    User
		wantID  int
		wantErr bool
	}{
		{"Создание нового пользователя", User{Username: "testuser", Email: "test@example.com", Password: "hashedpassword"}, 1, false},
		{"Создание пользователя с существующим email", User{Username: "duplicateuser", Email: "test@example.com", Password: "hashedpassword"}, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.wantErr {
				mockDB.EXPECT().CreateUser(gomock.Any(), tt.user).Return(tt.wantID, nil)
			} else {
				mockDB.EXPECT().CreateUser(gomock.Any(), tt.user).Return(0, assert.AnError)
			}

			gotID, err := mockDB.CreateUser(context.Background(), tt.user)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotID != tt.wantID {
				t.Errorf("CreateUser() = %v, want %v", gotID, tt.wantID)
			}
		})
	}
}

func TestDB_GetUserByEmail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := NewMockDBInterface(ctrl)

	tests := []struct {
		name    string
		email   string
		want    User
		wantErr bool
	}{
		{
			name:    "Получение существующего пользователя",
			email:   "test@example.com",
			want:    User{ID: 1, Username: "testuser", Email: "test@example.com", Password: "hashedpassword"},
			wantErr: false,
		},
		{
			name:    "Получение несуществующего пользователя",
			email:   "nonexistent@example.com",
			want:    User{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.wantErr {
				mockDB.EXPECT().GetUserByEmail(gomock.Any(), tt.email).Return(tt.want, nil)
			} else {
				mockDB.EXPECT().GetUserByEmail(gomock.Any(), tt.email).Return(User{}, assert.AnError)
			}

			got, err := mockDB.GetUserByEmail(context.Background(), tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetUserByEmail() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetUserByEmail() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDB_CreateReferralCode(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := NewMockDBInterface(ctrl)

	tests := []struct {
		name    string
		userID  int
		code    string
		expires int64
		wantErr bool
	}{
		{"Создание реферального кода", 1, "REF123", time.Now().Add(24 * time.Hour).Unix(), false},
		{"Создание реферального кода с истекшим временем", 1, "REF456", time.Now().Add(-24 * time.Hour).Unix(), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.wantErr {
				mockDB.EXPECT().CreateReferralCode(gomock.Any(), tt.userID, tt.code, tt.expires).Return(nil)
			} else {
				mockDB.EXPECT().CreateReferralCode(gomock.Any(), tt.userID, tt.code, tt.expires).Return(assert.AnError)
			}

			err := mockDB.CreateReferralCode(context.Background(), tt.userID, tt.code, tt.expires)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateReferralCode() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDB_DeleteReferralCode(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := NewMockDBInterface(ctrl)

	tests := []struct {
		name    string
		userID  int
		wantErr bool
	}{
		{"Удаление реферального кода", 1, false},
		{"Удаление несуществующего реферального кода", 999, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.EXPECT().DeleteReferralCode(gomock.Any(), tt.userID).Return(nil)

			err := mockDB.DeleteReferralCode(context.Background(), tt.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteReferralCode() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDB_GetReferralCodeByEmail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := NewMockDBInterface(ctrl)

	tests := []struct {
		name    string
		email   string
		want    ReferralCode
		wantErr bool
	}{
		{"Получение реферального кода по существующему email", "test@example.com", ReferralCode{UserID: 1, Code: "REF123"}, false},
		{"Получение реферального кода по несуществующему email", "nonexistent@example.com", ReferralCode{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.wantErr {
				mockDB.EXPECT().GetReferralCodeByEmail(gomock.Any(), tt.email).Return(tt.want, nil)
			} else {
				mockDB.EXPECT().GetReferralCodeByEmail(gomock.Any(), tt.email).Return(ReferralCode{}, assert.AnError)
			}

			got, err := mockDB.GetReferralCodeByEmail(context.Background(), tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetReferralCodeByEmail() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetReferralCodeByEmail() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDB_RegisterWithReferralCode(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := NewMockDBInterface(ctrl)

	tests := []struct {
		name         string
		referralCode string
		user         User
		wantErr      bool
	}{
		{"Регистрация с существующим реферальным кодом", "REF123", User{Username: "newuser", Email: "newuser@example.com", Password: "hashedpassword"}, false},
		{"Регистрация с несуществующим реферальным кодом", "NONEXISTENT", User{Username: "anotheruser", Email: "anotheruser@example.com", Password: "hashedpassword"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.wantErr {
				mockDB.EXPECT().RegisterWithReferralCode(gomock.Any(), tt.referralCode, tt.user).Return(nil)
			} else {
				mockDB.EXPECT().RegisterWithReferralCode(gomock.Any(), tt.referralCode, tt.user).Return(assert.AnError)
			}

			err := mockDB.RegisterWithReferralCode(context.Background(), tt.referralCode, tt.user)
			if (err != nil) != tt.wantErr {
				t.Errorf("RegisterWithReferralCode() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
