// storage/db.go
package storage

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/jackc/pgx"
	"github.com/jackc/pgx/v4/pgxpool"
)

// Интерфейс для работы с базой данных
type DBInterface interface {
	CreateUser(user User) (int, error)
	GetUserByEmail(email string) (User, error)
	CreateReferralCode(userID int, code string, expiresAt int64) error
	DeleteReferralCode(userID int) error
	GetReferralCodeByEmail(email string) (ReferralCode, error)
	GetReferralsByReferrerID(referrerID int) ([]User, error)
	RegisterWithReferralCode(referralCode string, user User) error
}

// Конфигурация БД
type DBConfig struct {
	Host     string `json:"host"`
	User     string `json:"user"`
	Password string `json:"password"`
	DBName   string `json:"dbname"`
	Port     int    `json:"port"`
	SSLMode  string `json:"sslmode"`
}

// База данных
type DB struct {
	pool *pgxpool.Pool
}

// Модель пользователя
type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"` // Хэшированный пароль
}

// Модель реферального кода
type ReferralCode struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	Code      string    `json:"code"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Конструктор для инициализации соединения с БД
func New(connstr string) (*DB, error) {
	if connstr == "" {
		return nil, errors.New("не указано подключение к БД")
	}
	pool, err := pgxpool.Connect(context.Background(), connstr)
	if err != nil {
		return nil, err
	}
	db := DB{
		pool: pool,
	}

	return &db, nil
}

// Создание пользователя
func (db *DB) CreateUser(user User) (int, error) {
	var userID int
	err := db.pool.QueryRow(context.Background(), `
		INSERT INTO users (username, email, password)
		VALUES ($1, $2, $3)
		RETURNING id`,
		user.Username,
		user.Email,
		user.Password,
	).Scan(&userID) // Получаем ID нового пользователя

	if err != nil {
		return 0, err // Возвращаем 0 и ошибку, если произошла ошибка
	}

	return userID, nil // Возвращаем ID и nil, если все прошло успешно
}

// Получение пользователя по email
func (db *DB) GetUserByEmail(email string) (User, error) {
	var user User
	err := db.pool.QueryRow(context.Background(), `
		SELECT id, username, email, password FROM users WHERE email = $1`, email).
		Scan(&user.ID, &user.Username, &user.Email, &user.Password)
	if err != nil {
		return User{}, err
	}
	return user, nil
}

// Создание реферального кода с проверкой на существующий код
func (db *DB) CreateReferralCode(userID int, code string, expiresAt int64) error {
	// Удаляем существующий активный код перед созданием нового
	if err := db.DeleteReferralCode(userID); err != nil {
		return err
	}

	_, err := db.pool.Exec(context.Background(), `
    INSERT INTO referral_codes (user_id, code, expires_at)
    VALUES ($1, $2, to_timestamp($3))`,
		userID,
		code,
		expiresAt,
	)
	return err
}

// Удаление реферального кода
func (db *DB) DeleteReferralCode(userID int) error {
	_, err := db.pool.Exec(context.Background(), `
		DELETE FROM referral_codes WHERE user_id = $1`,
		userID,
	)
	return err
}

// Получение реферального кода по email
func (db *DB) GetReferralCodeByEmail(email string) (ReferralCode, error) {
	var referralCode ReferralCode
	var userID int
	err := db.pool.QueryRow(context.Background(), `
		SELECT rc.id, rc.user_id, rc.code, rc.expires_at 
		FROM referral_codes rc 
		JOIN users u ON rc.user_id = u.id 
		WHERE u.email = $1`, email).
		Scan(&referralCode.ID, &userID, &referralCode.Code, &referralCode.ExpiresAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ReferralCode{}, errors.New("реферальный код не найден для данного email")
		}
		return ReferralCode{}, err
	}

	referralCode.UserID = userID
	return referralCode, nil
}

// Получение рефералов по ID реферера
func (db *DB) GetReferralsByReferrerID(referrerID int) ([]User, error) {
	rows, err := db.pool.Query(context.Background(), `
		SELECT u.id, u.username, u.email FROM referral_links rl
		JOIN users u ON rl.referee_id = u.id
		WHERE rl.referrer_id = $1`, referrerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var referrals []User
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Username, &user.Email); err != nil {
			return nil, err
		}
		referrals = append(referrals, user)
	}
	return referrals, rows.Err()
}

// В обработчике регистрации с реферальным кодом
func (db *DB) RegisterWithReferralCode(referralCode string, user User) error {
	// Проверка реферального кода
	var referrerID int
	var userID int
	err := db.pool.QueryRow(context.Background(), `
        SELECT user_id FROM referral_codes WHERE code = $1 AND expires_at > NOW()`, referralCode).
		Scan(&referrerID)
	if err != nil {
		log.Printf("Ошибка при проверке реферального кода: %v", err) // Логируем ошибку
		return err                                                   // Код недействителен
	}

	// Создание пользователя
	if userID, err = db.CreateUser(user); err != nil {
		log.Printf("Ошибка при создании пользователя: %v", err) // Логируем ошибку
		return err
	}

	// Создание записи о реферале
	_, err = db.pool.Exec(context.Background(), `
        INSERT INTO referral_links (referrer_id, referee_id) VALUES ($1, $2)`,
		referrerID,
		userID)
	return err
}
