package handlers

import (
	"CLIExpense/auth"
	"CLIExpense/middleware"
	"CLIExpense/models"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	ExpenseM models.ExpenseModel
	UserM    models.UserModel
}

func (h Handler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var input models.RegisterInput
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		http.Error(w, "Некорректный формат JSON", http.StatusBadRequest)
		return
	}
	response := models.ValidateUserData(input)
	if response != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		err = json.NewEncoder(w).Encode(response)
		if err != nil {
			http.Error(w, "Ошибка при кодировании файла", http.StatusBadRequest)
			return
		}
		return
	}
	id, err := h.UserM.CheckPassword(input.Email, input.Password)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		if appErr, ok := err.(models.AppError); ok {
			json.NewEncoder(w).Encode(map[string]string{"message": appErr.Message})
		} else {
			json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
		}
		return
	}

	token, err := auth.GenerateToken(id)
	if err != nil {
		http.Error(w, "Ошибка сервера", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(map[string]string{"token": token})

}

func (h Handler) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var input models.RegisterInput

	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		http.Error(w, "Некорректный формат JSON", http.StatusBadRequest)
		return
	}
	response := models.ValidateUserData(input)
	if response != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		err = json.NewEncoder(w).Encode(response)
		if err != nil {
			http.Error(w, "Ошибка при кодировании файла", http.StatusBadRequest)
			return
		}
		return
	}
	err = h.UserM.Register(input.Email, input.Password)
	if err != nil {
		var appErr models.AppError
		if errors.As(err, &appErr) {
			http.Error(w, appErr.Message, appErr.Status)
		} else {
			http.Error(w, "внутреняя ошибка сервера", http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Пользователь успешно зарегистрирован!"})
}

// ! Стартовая страница
func HelloHandler(w http.ResponseWriter, r *http.Request) {
	models.ExpenseMu.Lock()
	defer models.ExpenseMu.Unlock()
	fmt.Fprint(w, "Введите запросы в строку поиска чтобы продолжить\n/expenses - посмотреть траты\n/add - добавить тарату\n/total - показать общую сумму трат")
}

// ! Созадние гет запроса, просим показать записи которые уже есть
func (h Handler) ExpensesHandler(w http.ResponseWriter, r *http.Request) {
	// Достаем id пользователя, который совершил этот запрос!
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Ошибка авторизации", http.StatusUnauthorized)
		return
	}

	pages := 1
	pagesStr := r.URL.Query().Get("page")
	if pagesStr != "" {
		var err error
		pages, err = strconv.Atoi(pagesStr)
		if err != nil || pages <= 0 {
			http.Error(w, "Ошибка параметра page", http.StatusBadRequest)
			return

		}
	}

	limit := 10
	limitStr := r.URL.Query().Get("limit")
	if limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			http.Error(w, "Ошибка параемтра limit", http.StatusBadRequest)
			return

		}
	}

	offset := (pages - 1) * limit
	// Передаем userID в базу данных, чтобы получить траты ТОЛЬКО ЭТОГО пользователя!
	expenses, err := h.ExpenseM.GetAllExpenses(limit, offset, userID)
	if err != nil {
		var appErr models.AppError

		if errors.As(err, &appErr) {
			http.Error(w, appErr.Message, appErr.Status)
		} else {
			http.Error(w, "Непредвиденная ошибка", http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(expenses)
}

// !Создание пост запроса, добавление новой траты
func (h Handler) ExpensesCreateHandler(w http.ResponseWriter, r *http.Request) {
	//? для отправки запроса - curl -Method Post -Uri "http://localhost:8080/add" -Header @{"Content-Type"="application/json"} -Body '{"name":"Pizza","price":850}'
	var newExpense models.Expense
	err := json.NewDecoder(r.Body).Decode(&newExpense)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response := models.ValidateExpense(newExpense)
	if response != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		err = json.NewEncoder(w).Encode(response)
		if err != nil {
			http.Error(w, "Ошибка при кодировании файла.", http.StatusBadRequest)
			return
		}
		return
	}
	userID := r.Context().Value(middleware.UserIDKey).(int)
	err = h.ExpenseM.InsertExpense(newExpense, userID)
	if err != nil {
		var appErr models.AppError

		if errors.As(err, &appErr) {
			http.Error(w, appErr.Message, appErr.Status)
		} else {
			http.Error(w, "Непредвиденная ошибка", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusCreated)
	fmt.Fprint(w, "Трата добавлена")
}

// ! Создание запроса одной траты по ID
func (h Handler) GetExpenseByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id < 0 {
		http.Error(w, "Отправлены не верные данные", http.StatusBadRequest)
		return
	}
	userID := r.Context().Value(middleware.UserIDKey).(int)

	res, err := h.ExpenseM.GetOneExpense(id, userID)
	if err != nil {
		var appErr models.AppError

		if errors.As(err, &appErr) {
			http.Error(w, appErr.Message, appErr.Status)
		} else {
			http.Error(w, "Непредвиденная ошибка", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(res)

	if err != nil {
		http.Error(w, "Ошибка кодирования", http.StatusBadRequest)
		return
	}
}

// ! Создание DLEATE запроса
func (h Handler) ExpensesDel(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id") // Достанет то, что попало в {id}
	id, err := strconv.Atoi(idStr)
	if err != nil || id < 0 {
		http.Error(w, "Отправлены не верные данные", http.StatusBadRequest)
		return
	}
	userID := r.Context().Value(middleware.UserIDKey).(int)

	err = h.ExpenseM.DeleteFromID(id, userID)
	if err != nil {
		var appErr models.AppError

		if errors.As(err, &appErr) {
			http.Error(w, appErr.Message, appErr.Status)
		} else {
			http.Error(w, "Непрежвиденная ошибка", http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]string{"message": "Элемент удален успешно!"}
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, "Ошибка при кодировании файла.", http.StatusBadRequest)
		return
	}
}

// ! Создание гет запроса, получение общей суммы
func (h Handler) TotalHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(int)

	total, err := h.ExpenseM.TotalFromPrice(userID)
	if err != nil {
		http.Error(w, "Внутриняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]float64{"total_price": total}

	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, "Ошибка при кодировании.", http.StatusInternalServerError)
		return
	}
}
