package middlware

import (
	"context"
	"fmt"
	"net/http"

	"gorefer.go/pkg/auth"
)

type contextKey string

const (
	UserKey contextKey = "username"
)

// TokenAuthMiddleware проверяет токен и добавляет пользователя в контекст
func TokenAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString := r.Header.Get("Authorization")
		if tokenString == "" || len(tokenString) < len("Bearer ") {
			http.Error(w, "Токен не предоставлен", http.StatusUnauthorized)
			return
		}

		tokenString = tokenString[len("Bearer "):]

		username, err := auth.ValidateToken(tokenString)
		if err != nil {
			http.Error(w, "Недействительный токен", http.StatusUnauthorized)
			fmt.Println("Ошибка при проверке токена:", err)
			return
		}

		ctx := context.WithValue(r.Context(), UserKey, username)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}
