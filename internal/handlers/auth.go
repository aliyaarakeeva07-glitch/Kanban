package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var jwtKey = []byte("my_secret_key")

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type Claims struct {
	UserID int `json:"user_id"`
	jwt.RegisteredClaims
}

type AuthHandler struct {
	DB *sql.DB
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest

	// читаем JSON
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// ищем пользователя
	var id int
	var passwordHash string

	err = h.DB.QueryRow(
		"SELECT id, password_hash FROM users WHERE email=$1",
		req.Email,
	).Scan(&id, &passwordHash)

	if err != nil {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	// проверяем пароль
	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password))
	if err != nil {
		http.Error(w, "Wrong password", http.StatusUnauthorized)
		return
	}

	// создаём токен
	expirationTime := time.Now().Add(24 * time.Hour)

	claims := &Claims{
		UserID: id,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		http.Error(w, "Could not create token", http.StatusInternalServerError)
		return
	}

	// ответ
	json.NewEncoder(w).Encode(map[string]string{
		"token": tokenString,
	})
}
