package middleware

import (
	"CLIExpense/auth"
	"context"
	"log"
	"net/http"
	"strings"
	"time"
)

func TimingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		next.ServeHTTP(w, r)

		duration := time.Since(start)

		log.Printf("[%s] - %s : %v", r.Method, r.URL.Path, duration)
	})

}

// Создаем уникальный тип для ключа контекста, чтобы избежать конфликтов
type contextKey string

const UserIDKey contextKey = "UserID"

func AuthMIddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Достаем заголовок Authorization
		authHeader := r.Header.Get("Authorization")
		if strings.TrimSpace(authHeader) == "" {
			http.Error(w, "Отсутствует заголовок Authorization", http.StatusUnauthorized)
			return
		}
		// 2. Обычно токен присылают в формате: "Bearer <токен>"
		// Нам нужно отрезать слово "Bearer "
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Использован не верный формат заголовка. Используйте 'Bearer <токен>'", http.StatusUnauthorized)
			return
		}
		tokenStr := parts[1]
		// 3. Валидируем токен
		userID, err := auth.ValidateToken(tokenStr)
		if err != nil {
			http.Error(w, "Неверный или просроченный токен", http.StatusUnauthorized)
			return
		}
		// 4. МЫ ВНУТРИ! Токен верный.
		// Кладем userID в "кармашек" (контекст) запроса
		ctx := context.WithValue(r.Context(), UserIDKey, userID)

		// Обновляем запрос, добавив в него наш новый контекст
		r = r.WithContext(ctx)

		// 5. Передаем управление следующему хендлеру по конвейеру
		next.ServeHTTP(w, r)
	})
}
