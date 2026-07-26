package models

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

type Expense struct {
	Name      string     `json:"name"`     //поле название расхода, тип данных строка
	Price     float64    `json:"price"`    //поле стоимость этого расходо с типом данных float64 число с плавающей точкой
	Category  string     `json:"category"` //поле обобщённой категории
	ID        int        `json:"id"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"` // 2. Указатель! omitempty скроет поле в JSON, если оно nil
}

type Budget struct {
	Tracker []Expense
}

type ExpenseModel struct {
	DB *sql.DB
}

var (
	ExpenseMu    sync.Mutex
	GlobalBudget Budget
)

type RegisterInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

const Filename = "my_expenses.json"

func (m ExpenseModel) TotalFromPrice(userID int) (float64, error) {
	var total float64
	err := m.DB.QueryRow("SELECT COALESCE(SUM(price),0) FROM expenses WHERE user_id = $1 AND deleted_at IS NULL", userID).Scan(&total) //создание запрос который вернёт одну строку, COALESCE(SUM(price),0) - говорит что если первое значение NULL возми второе значение
	if err != nil {
		return 0, err
	}

	return total, nil

}

func (m ExpenseModel) GetOneExpense(id int, userID int) (Expense, error) {
	query := "SELECT id,name,price,category,deleted_at FROM expenses WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL"
	var e Expense
	err := m.DB.QueryRow(query, id, userID).Scan(&e.ID, &e.Name, &e.Price, &e.Category, &e.DeletedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Expense{}, AppError{
				Err:     err,
				Message: "Трата с таким id не найдена или у вас нет прав на её просмотр",
				Status:  http.StatusNotFound,
			}
		}
		return Expense{}, AppError{
			Err:     err,
			Message: "Внутреняя ошибка базы данных",
			Status:  http.StatusInternalServerError,
		}

	}
	return e, nil
}

func (m ExpenseModel) DeleteFromID(id, userID int) error {
	query := "UPDATE expenses SET deleted_at = NOW() WHERE id=$1 AND user_id = $2 AND deleted_at IS NULL" //удаление строки по id ипользуя плейсхолдер(подставное значение,защита от sql инъекций)
	// ! AND deleted_at IS NULL — это защита от повторного удаления уже удаленного элемента
	res, err := m.DB.Exec(query, id, userID)
	// res хранит результат удаления, объект sql.Result
	if err != nil {
		return AppError{
			Err:     err,
			Message: "Ошибка сервера",
			Status:  http.StatusInternalServerError,
		}
	}

	count, err := res.RowsAffected() // проверяем количество затронутых строк
	if err != nil {
		return AppError{
			Err:     err,
			Message: "Ошибка сервера",
			Status:  http.StatusInternalServerError,
		}
	}

	if count == 0 {
		return AppError{
			Err:     err,
			Message: "Траты с таким id не существует или у вас нет прав на её удалени",
			Status:  http.StatusNotFound,
		}
	}

	return nil
}

func (m ExpenseModel) GetAllExpenses(limit, offset, userID int) ([]Expense, error) {
	rows, err := m.DB.Query("SELECT id, name, price, category,deleted_at FROM expenses WHERE user_id = $1 AND deleted_at IS NULL LIMIT $2 OFFSET $3", userID, limit, offset) //ищем каждое поле в таблице

	if err != nil {
		return nil, AppError{
			Err:     err,
			Message: "Ошибка сервера",
			Status:  http.StatusInternalServerError,
		}
	}
	defer rows.Close()

	var expenses []Expense

	for rows.Next() {
		var e Expense
		err := rows.Scan(&e.ID, &e.Name, &e.Price, &e.Category, &e.DeletedAt) //сканируем каждую ячейку
		if err != nil {
			return nil, AppError{
				Err:     err,
				Message: "Ошибка сервера",
				Status:  http.StatusInternalServerError,
			}
		}
		expenses = append(expenses, e)
	}
	return expenses, nil
}

// ! Запись данных в базу данных
func (m ExpenseModel) InsertExpense(e Expense, userID int) error {
	query := "INSERT INTO expenses (name, price, category, user_id) VALUES ($1, $2, $3, $4)"
	_, err := m.DB.Exec(query, e.Name, e.Price, e.Category, userID)
	if err != nil {
		return AppError{
			Err:     err,
			Message: "Не удалось сохранить трату в базе данных",
			Status:  http.StatusInternalServerError,
		}
	}

	return nil
}

func InitDB() (*sql.DB, error) { //! Создание подключения к базе данных
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname) //? инициализация, меняется только пороль и имя базы

	var err error
	for i := 0; i <= 5; i++ {
		DB, err := sql.Open("postgres", connStr) //!подключение
		if err == nil {
			err = DB.Ping() //!Проверка подключения к бд
			if err == nil {
				break
			}
			log.Printf("Попытка %d: бд ещё не готова, ждём...", i)
			time.Sleep(2 * time.Second)
		}
		if err != nil {
			log.Fatalf("Не удадлсь подключиться к бд после 5 пыпыток: %v", err)
		}

	}

	query := ` 
	CREATE TABLE IF NOT EXISTS expenses (
		id SERIAL PRIMARY KEY,
		name TEXT NOT NULL,
		price DOUBLE PRECISION NOT NULL,
		category TEXT NOT NULL,
		deleted_at TIMESTAMP DEFAULT NULL
	);` //? Создание таблицы даных в базе данных

	_, err = DB.Exec(query)
	if err != nil {
		return nil, fmt.Errorf("не удалось создать таблицу - %v ", err)
	}
	fmt.Println("Успешное подключение к PostgreSQL")
	return DB, nil
}

func (b *Budget) LoadFromFile(filename string) error { //! этот процесс называется десериализация, сбор среза байт в читаемые строки
	if filename == "" { //! Проверка не пустое ли поле на пришло
		return fmt.Errorf("Ошибка, имя файла пустое поле!")
	}

	res, err := os.ReadFile(filename)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("Произошла ошибка при чтении из файла!")
	}

	err = json.Unmarshal(res, &b.Tracker)
	if err != nil {
		return fmt.Errorf("Произошла ошибка при десериализации файла!")
	}
	return nil
}

func (b *Budget) SaveToFile(filename string) error { //todo этот процесс называется сериализация, создание из строки срез байтов и запись в файл
	if filename == "" { //! Проверка не пустое ли поле на пришло
		return fmt.Errorf("Ошибка, имя файла пустое поле!")
	}
	res, err := json.Marshal(b.Tracker) //!превращение списка трат в []byte, обязательно две переменные для результат и отработки ошибок
	if err != nil {
		return fmt.Errorf("Произошла ошибка при получении байт файла: %w", err)
	}
	err = os.WriteFile(filename, res, 0644) //! Запись данных в файл, обязательная переменная для проверки ошибок

	if err != nil {
		return fmt.Errorf("Ошибка при записи данных в файл!")
	}

	return nil

}

func parseNumber(promt string, reader *bufio.Reader) (float64, error) {
	fmt.Print(promt)
	strNum, err := reader.ReadString('\n')
	if err != nil {
		return 0, fmt.Errorf("Произошла ошибка: %w", err)
	}
	strNum = strings.TrimSpace(strNum)
	num, err := strconv.ParseFloat(strNum, 64)
	return num, err
}

func (b *Budget) AddExpense(reader *bufio.Reader) error { //из main приходит ввод имя и число, добавляем это в список трат и возвращаем из функции, ошибка при неверном вводе //! исправление : Функция должна принимать текущий список трат, добавлять туда новую и возвращать обновленный список
	// ? ИЗМЕНЕНИЕ 2 удалены price и name из арнументов функции, ввод будет внутри функции, добавлен reader

	fmt.Print("Введите название траты: ")
	exp, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("Произошла ошибка: %w", err)
	}
	exp = strings.TrimSpace(exp)
	if exp == "" {
		return fmt.Errorf("Название траты не может быть пустым")
	}

	item, err := parseNumber("Введите стоимоть:", reader)
	if err != nil {
		return fmt.Errorf("Произошла ошибка: %w", err)
	}

	newExpense := Expense{
		Name:  exp,
		Price: item,
	}
	b.Tracker = append(b.Tracker, newExpense)
	return nil

}

func (b *Budget) String() string { //получам список трат по которому пройдемся циклом, корректно отработаем и вернём стороку с тратой, так же функция может вренуть ошибку при пустом списке//! исправление:Для ShowExpense: если список пустой, функция может вернуть пустую строку (или специальное сообщение), а main уже решит, как это красиво напечатать.

	if len(b.Tracker) == 0 {
		return ""
	}
	var res string
	for _, item := range b.Tracker {
		res += fmt.Sprintf("- %s : %.2f руб.\n", item.Name, item.Price)
	}
	return res
}

func (b *Budget) TotalPrice() float64 { // получаем список трат, берём из него только Price считаем сумму и возращаем её, так же отрабаатвыем ошибки пустого списка //! Исправление: В Go error возвращают тогда, когда произошло что-то непредвиденное, ломающее логику (сломалась база данных, пользователь ввел буквы вместо цифр).Но то, что у пользователя пока нет трат — это не ошибка программы, это нормальная жизненная ситуация (он только что скачал приложение).Как сделать лучше: * Для TotalPrice: если список пустой, функция может просто вернуть 0.0 без всяких ошибок. Это логично: нет трат — сумма ноль.
	total := 0.0

	for _, item := range b.Tracker {
		total += item.Price
	}
	return total
}

// ! валидация ошибок
func ValidateExpense(expense Expense) map[string]string {
	errors := make(map[string]string)
	if strings.TrimSpace(expense.Name) == "" {
		errors["name"] = "имя не может быть пустым"
	}
	if expense.Price <= 0 {
		errors["price"] = "цена не может быть отрицательной или равной 0"
	}
	if strings.TrimSpace(expense.Category) == "" {
		errors["category"] = "поле категории не должно пустовать"
	}
	if len(errors) > 0 {
		return errors
	}
	return nil
}

func ValidateUserData(registerInput RegisterInput) map[string]string {
	errors := make(map[string]string)
	if strings.TrimSpace(registerInput.Email) == "" {
		errors["email"] = "поле email не может быть пустым"
	}
	if strings.TrimSpace(registerInput.Password) == "" {
		errors["password"] = "поле password не может быть пустым"
	}
	if len(registerInput.Password) < 6 {
		errors["password"] = "пароль должен быть больше 6 символов"
	}
	if len(errors) > 0 {
		return errors
	}
	return nil
}
