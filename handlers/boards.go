package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"kanban/middleware"

	"github.com/gorilla/mux"
)

type Board struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type BoardHandler struct {
	DB *sql.DB
}

// 🔹 GET /boards
func (h *BoardHandler) GetBoards(w http.ResponseWriter, r *http.Request) {

	userID := r.Context().Value(middleware.UserIDKey).(int)

	rows, err := h.DB.Query(`
		SELECT DISTINCT b.id, b.title, b.description
		FROM boards b
		LEFT JOIN board_members bm ON bm.board_id = b.id
		WHERE (b.owner_id=$1 OR bm.user_id=$1)
		AND b.deleted_at IS NULL
	`, userID)

	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	boards := []Board{}

	for rows.Next() {
		var b Board

		err := rows.Scan(&b.ID, &b.Title, &b.Description)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		boards = append(boards, b)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(boards)
}

// 🔹 POST /boards
func (h *BoardHandler) CreateBoard(w http.ResponseWriter, r *http.Request) {

	userID := r.Context().Value(middleware.UserIDKey).(int)

	var b Board
	err := json.NewDecoder(r.Body).Decode(&b)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	var boardID int

	err = h.DB.QueryRow(`
		INSERT INTO boards (title, description, owner_id)
		VALUES ($1, $2, $3)
		RETURNING id
	`, b.Title, b.Description, userID).Scan(&boardID)

	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// добавляем owner в members
	_, _ = h.DB.Exec(`
		INSERT INTO board_members (board_id, user_id, role)
		VALUES ($1, $2, 'owner')
	`, boardID, userID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{
		"id": boardID,
	})
}

// 🔹 DELETE /boards/:id (soft delete)
func (h *BoardHandler) DeleteBoard(w http.ResponseWriter, r *http.Request) {

	userID := r.Context().Value(middleware.UserIDKey).(int)

	vars := mux.Vars(r)
	boardID := vars["id"]

	var ownerID int

	err := h.DB.QueryRow(`
		SELECT owner_id FROM boards WHERE id=$1
	`, boardID).Scan(&ownerID)

	if err != nil {
		http.Error(w, "Board not found", 404)
		return
	}

	// проверка owner
	if ownerID != userID {
		http.Error(w, "Only owner can delete", 403)
		return
	}

	_, err = h.DB.Exec(`
		UPDATE boards SET deleted_at = NOW() WHERE id=$1
	`, boardID)

	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	_, _ = w.Write([]byte("Board deleted"))
}

// 🔹 GET /boards/:id/members
func (h *BoardHandler) GetMembers(w http.ResponseWriter, r *http.Request) {

	userID := r.Context().Value(middleware.UserIDKey).(int)
	vars := mux.Vars(r)
	boardID := vars["id"]

	// 🔐 проверка доступа (owner или member)
	var exists int
	err := h.DB.QueryRow(`
		SELECT 1
		FROM boards b
		LEFT JOIN board_members bm ON bm.board_id = b.id
		WHERE b.id=$1 AND (b.owner_id=$2 OR bm.user_id=$2)
	`, boardID, userID).Scan(&exists)

	if err != nil {
		http.Error(w, "Access denied", 403)
		return
	}

	// 📌 получаем участников
	rows, err := h.DB.Query(`
		SELECT u.id, u.name, bm.role
		FROM board_members bm
		JOIN users u ON u.id = bm.user_id
		WHERE bm.board_id=$1
	`, boardID)

	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	type Member struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
		Role string `json:"role"`
	}

	members := []Member{}

	for rows.Next() {
		var m Member
		err := rows.Scan(&m.ID, &m.Name, &m.Role)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		members = append(members, m)
	}

	_ = json.NewEncoder(w).Encode(members)
}
