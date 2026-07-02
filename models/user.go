package models

import (
	"database/sql"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID        int
	Email     string
	Password  string
	CreatedAt *time.Time
}

type UserModel struct {
	DB *sql.DB
}

func (m UserModel) Register(email, password string) error {
	// хешируем пароль
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return AppError{
			Err:     err,
			Message: "Ошибка хеширования пороля",
		}
	}

	// 2. Пишем SQL-запрос для вставки в таблицу users
	query := "INSERT INTO users (email,password_hash) VALUES ($1,$2)"

	// 3. Выполняем запрос
	_, err = m.DB.Exec(query, email, string(hashedPassword))
	if err != nil {
		// Тут есть важный нюанс! Если email уже существует, Postgres вернет ошибку уникальности (unique violation).
		// Пока для простоты вернем общую ошибку сервера, но держи это в голове.
		return AppError{
			Err:     err,
			Message: "Не удалось зарегестрировать пользователя. Возможно такой email уже занят.",
			Status:  http.StatusInternalServerError,
		}
	}

	return nil
}

func (m UserModel) CheckPassword(email, password string) (int, error) {

	var id int
	var passwordHash string

	query := "SELECT id, password_hash FROM users WHERE email = $1"
	err := m.DB.QueryRow(query, email).Scan(&id, &passwordHash)
	if err != nil {
		return 0, AppError{
			Err:     err,
			Message: "Неверный логин или пароль",
			Status:  http.StatusUnauthorized,
		}
	}
	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
	if err != nil {
		return 0, AppError{
			Err:     err,
			Message: "Неверный логин или пароль",
			Status:  http.StatusUnauthorized,
		}
	}
	return id, nil
}
