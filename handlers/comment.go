package handlers

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"net/http"
	"strconv"
	"time"
)

// Comment соответствует твоей таблице: id, task_id, user_id, text, created_at, deleted_at
type Comment struct {
	ID        int        `json:"id"`
	TaskID    int        `json:"task_id"`
	UserID    int        `json:"user_id"`
	Content   string     `json:"content"`
	CreatedAt time.Time  `json:"created_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

// GET /comments?task_id=1
func (h *TaskHandler) GetComments(w http.ResponseWriter, r *http.Request) {
	taskIDStr := r.URL.Query().Get("task_id")
	taskID, _ := strconv.Atoi(taskIDStr)

	// Выбираем только те, что НЕ удалены (deleted_at IS NULL)
	query := `
		SELECT id, task_id, user_id, content, created_at 
		FROM comments 
		WHERE task_id = $1 AND deleted_at IS NULL 
		ORDER BY created_at ASC`

	rows, err := h.DB.Query(query, taskID)
	if err != nil {
		http.Error(w, "Ошибка БД: "+err.Error(), 500)
		return
	}
	defer rows.Close()

	comments := []Comment{}
	for rows.Next() {
		var c Comment
		if err := rows.Scan(&c.ID, &c.TaskID, &c.UserID, &c.Content, &c.CreatedAt); err != nil {
			http.Error(w, "Ошибка сканирования: "+err.Error(), 500)
			return
		}
		comments = append(comments, c)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(comments)
}

// POST /comments
func (h *TaskHandler) CreateComment(w http.ResponseWriter, r *http.Request) {
	var c Comment
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, "Ошибка JSON", 400)
		return
	}

	query := `
		INSERT INTO comments (task_id, user_id, content) 
		VALUES ($1, $2, $3) 
		RETURNING id, created_at`

	err := h.DB.QueryRow(query, c.TaskID, c.UserID, c.Content).Scan(&c.ID, &c.CreatedAt)
	if err != nil {
		http.Error(w, "Ошибка вставки: "+err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(c)
}

// DELETE /comments/{id} - Мягкое удаление
func (h *TaskHandler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	commentID, _ := strconv.Atoi(idStr)

	// Ставим метку времени вместо удаления
	query := "UPDATE comments SET deleted_at = NOW() WHERE id = $1"

	result, err := h.DB.Exec(query, commentID)
	if err != nil {
		http.Error(w, "Ошибка удаления: "+err.Error(), 500)
		return
	}

	count, _ := result.RowsAffected()
	if count == 0 {
		http.Error(w, "Комментарий не найден", 404)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
