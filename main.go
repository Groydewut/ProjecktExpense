package main

import (
	"fmt"
	"log"
	"mime"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"

	// Убедись, что эти пути импортов совпадают с твоим go.mod
	"CLIExpense/handlers"
	"CLIExpense/middleware"
	"CLIExpense/models"
)

func main() {
	mime.AddExtensionType(".css", "text/css; charset=utf-8")
	mime.AddExtensionType(".js", "application/javascript; charset=utf-8")

	err := godotenv.Load()
	if err != nil {
		log.Println("Предупреждение: файл .env не найден, используем системные переменные окружения")
	}

	db, err := models.InitDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// 🔥 ЗАЩИТА ОТ ПАНИКИ: Если вдруг InitDB() сбойнул внутри, но забыл вернуть err,
	// эта проверка не даст серверу упасть с загадочным "nil pointer" в базе данных.
	if db == nil {
		log.Fatal("Критическая ошибка: Переменная базы данных равна nil! Проверь настройки подключения в models.InitDB()")
	}

	myModel := models.ExpenseModel{DB: db}
	userModel := models.UserModel{DB: db}
	myHandler := handlers.Handler{
		ExpenseM: myModel,
		UserM:    userModel,
	}

	//!Создание главного роутера
	r := chi.NewRouter()
	//!подключаем логер middleware, после этого каждый запрос записывается в консоль
	r.Use(middleware.TimingMiddleware)

	//!определяем маршруты(rest API)

	r.Post("/register", myHandler.RegisterHandler)
	r.Post("/login", myHandler.LoginHandler)

	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware)

		r.Get("/expenses", myHandler.ExpensesHandler)
		r.Get("/expenses/{id}", myHandler.GetExpenseByID)
		r.Post("/expenses", myHandler.ExpensesCreateHandler)
		r.Get("/total", myHandler.TotalHandler)

		// 🛠️ ИСПРАВЛЕНО: Привели к истинному REST-стандарту.
		// Вместо /delete/{id} сделали метод DELETE на адрес /expenses/{id}.
		// Это убирает ошибку 405 Method Not Allowed на фронтенде.
		r.Delete("/expenses/{id}", myHandler.ExpensesDel)
	})

	fileserver := http.FileServer(http.Dir("./frontend"))
	// Перехватываем любые GET-запросы, которые не подошли под верхние правила (например, /, /index.html, /app.js)
	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		fileserver.ServeHTTP(w, r)
	})

	// Поправил опечатку "Сревер запущен" :)
	fmt.Println("Сервер запущен на http://localhost:8080")

	log.Fatal(http.ListenAndServe(":8080", r))
}
