package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"kanban/middleware"

	"github.com/gorilla/mux"
)

type Column struct {
	ID           int        `json:"id"`
	BoardID      int        `json:"board_id"`
	Title        string     `json:"title"`
	Position     int        `json:"position"`
	LastPosition int        `json:"last_position"`
	DeletedAt    *time.Time `json:"deleted_at"` // Используем указатель, так как может быть NULL
}

type ColumnHandler struct {
	DB *sql.DB
}

// 🔹 GET /columns — Получить список активных колонок
func (h *ColumnHandler) GetColumns(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID := r.Context().Value(middleware.UserIDKey).(int)

	boardIDStr := r.URL.Query().Get("board_id")
	boardID, err := strconv.Atoi(boardIDStr)
	if err != nil {
		http.Error(w, "Неверный board_id", http.StatusBadRequest)
		return
	}

	// 🔐 Проверка доступа к доске
	var exists bool
	err = h.DB.QueryRow(`
       SELECT EXISTS(
          SELECT 1 FROM board_members
          WHERE board_id=$1 AND user_id=$2
       )
    `, boardID, userID).Scan(&exists)

	if err != nil || !exists {
		http.Error(w, "Доступ запрещен", http.StatusForbidden)
		return
	}

	rows, err := h.DB.Query(`
       SELECT id, board_id, title, position
       FROM columns
       WHERE board_id=$1 AND deleted_at IS NULL
       ORDER BY position
    `, boardID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var columns []Column
	for rows.Next() {
		var c Column
		rows.Scan(&c.ID, &c.BoardID, &c.Title, &c.Position)
		columns = append(columns, c)
	}

	json.NewEncoder(w).Encode(columns)
}

// 🔹 POST /columns — Создать колонку
func (h *ColumnHandler) CreateColumn(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(int)

	var c Column
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, "Ошибка в JSON", http.StatusBadRequest)
		return
	}

	// 🔐 Проверка доступа
	var exists bool
	h.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM board_members WHERE board_id=$1 AND user_id=$2)`,
		c.BoardID, userID).Scan(&exists)

	if !exists {
		http.Error(w, "Доступ запрещен", http.StatusForbidden)
		return
	}

	// Находим позицию
	var lastPos int
	h.DB.QueryRow(`SELECT COALESCE(MAX(position), 0) FROM columns WHERE board_id=$1`, c.BoardID).Scan(&lastPos)

	err := h.DB.QueryRow(`
       INSERT INTO columns (board_id, title, position)
       VALUES ($1, $2, $3) RETURNING id
    `, c.BoardID, c.Title, lastPos+1).Scan(&c.ID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(c)
}

// 🔹 DELETE /columns/{id} — Мягкое удаление (в корзину)
func (h *ColumnHandler) DeleteColumn(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(int)
	vars := mux.Vars(r)
	columnID := vars["id"]

	var boardID, position int
	err := h.DB.QueryRow(`SELECT board_id, position FROM columns WHERE id=$1`, columnID).Scan(&boardID, &position)
	if err != nil {
		http.Error(w, "Колонка не найдена", http.StatusNotFound)
		return
	}

	// 🔐 Проверка (только участники доски могут удалять)
	var exists bool
	h.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM board_members WHERE board_id=$1 AND user_id=$2)`, boardID, userID).Scan(&exists)
	if !exists {
		http.Error(w, "Доступ запрещен", http.StatusForbidden)
		return
	}

	// Сохраняем позицию и помечаем как удаленную
	_, err = h.DB.Exec(`
       UPDATE columns
       SET deleted_at=NOW(), last_position=position, position=-1
       WHERE id=$1
    `, columnID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte("Column moved to trash"))
}

// 🔹 PATCH /columns/{id}/restore — Восстановление из корзины
func (h *ColumnHandler) RestoreColumn(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(int)
	vars := mux.Vars(r)
	columnID := vars["id"]

	var boardID, lastPos int
	err := h.DB.QueryRow(`SELECT board_id, last_position FROM columns WHERE id=$1`, columnID).Scan(&boardID, &lastPos)
	if err != nil {
		http.Error(w, "Колонка не найдена", http.StatusNotFound)
		return
	}

	// Восстанавливаем позицию
	_, err = h.DB.Exec(`
		UPDATE columns 
		SET position = last_position, deleted_at = NULL 
		WHERE id = $1`, columnID)

	if err != nil {
		http.Error(w, "Ошибка восстановления", http.StatusInternalServerError)
		return
	}

	_ = userID // используем переменную, чтобы не было ошибки "not used"
	w.Write([]byte("Column restored"))
}
