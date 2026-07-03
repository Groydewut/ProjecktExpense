package main

import (
	"CLIExpense/handlers"
	"CLIExpense/middleware"
	"CLIExpense/models"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
)

// todo ВЕБ СЕРВЕР
func main() {

	err := godotenv.Load()
	if err != nil {
		log.Println("Предупреждение: файл .env не найден, используем системные переменные окружения")
	}

	db, err := models.InitDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	myModel := models.ExpenseModel{DB: db}
	myHandler := handlers.Handler{ExpenseM: myModel}

	//!Создание главного роутера
	r := chi.NewRouter()
	//!подключаем логер middleware, после этого каждый запрос записывается в консоль
	r.Use(middleware.TimingMiddleware)

	//!определяем маршруты(rest API)

	r.Get("/", handlers.HelloHandler)
	r.Post("/register", myHandler.RegisterHandler)
	r.Post("/login", myHandler.LoginHandler)

	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMIddleware)

		r.Get("/expenses", myHandler.ExpensesHandler)
		r.Get("/expenses/{id}", myHandler.GetExpenseByID)
		r.Post("/add", myHandler.ExpensesCreateHandler)
		r.Get("/total", myHandler.TotalHandler)
		// Магия chi: красивый URL-параметр {id} вместо ?id=...
		r.Delete("/delete/{id}", myHandler.ExpensesDel)
	})

	fmt.Println("Сревер запущен на http://localhost:8080 ")

	log.Fatal(http.ListenAndServe(":8080", r))

}
