package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type Task struct {
	ID          int     `json:"id"`
	ColumnID    int     `json:"column_id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Priority    string  `json:"priority"`
	Position    int     `json:"position"`
	DeletedAt   *string `json:"deleted_at"`
	ArchivedAt  *string `json:"archived_at"`
	DoneAt      *string `json:"done_at"`
}

type TaskHandler struct {
	DB *sql.DB
}

// Вспомогательная функция для отправки JSON-ошибок
func sendError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

const doneColumnID = 3

// 🔹 GET /tasks?column_id=1&search=текст
func (h *TaskHandler) GetTasks(w http.ResponseWriter, r *http.Request) {
	colIDStr := r.URL.Query().Get("column_id")
	searchQuery := r.URL.Query().Get("search")

	colID, err := strconv.Atoi(colIDStr)
	if err != nil {
		sendError(w, "Некорректный ID колонки", http.StatusBadRequest)
		return
	}

	// Безопасный запрос: параметры передаются отдельно через $1 и $2
	query := `
       SELECT id, column_id, title, description, priority, position, done_at 
       FROM tasks 
       WHERE column_id=$1 
         AND deleted_at IS NULL 
         AND archived_at IS NULL 
         AND title ILIKE $2
       ORDER BY position ASC`

	rows, err := h.DB.Query(query, colID, "%"+searchQuery+"%")
	if err != nil {
		sendError(w, "Ошибка базы данных: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	tasks := []Task{}
	for rows.Next() {
		var t Task
		err := rows.Scan(&t.ID, &t.ColumnID, &t.Title, &t.Description, &t.Priority, &t.Position, &t.DoneAt)
		if err != nil {
			continue
		}
		tasks = append(tasks, t)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

// 🔹 POST /tasks — Создать задачу
func (h *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	var t Task
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		sendError(w, "Некорректный формат JSON", http.StatusBadRequest)
		return
	}

	if t.Title == "" {
		sendError(w, "Название задачи не может быть пустым", http.StatusBadRequest)
		return
	}

	var maxPos int
	h.DB.QueryRow("SELECT COALESCE(MAX(position), 0) FROM tasks WHERE column_id=$1", t.ColumnID).Scan(&maxPos)

	err := h.DB.QueryRow(`
       INSERT INTO tasks (column_id, title, description, priority, position)
       VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		t.ColumnID, t.Title, t.Description, t.Priority, maxPos+1).Scan(&t.ID)

	if err != nil {
		sendError(w, "Не удалось создать задачу", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(t)
}

// 🔹 DELETE /tasks/{id} — Мягкое удаление
func (h *TaskHandler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	result, err := h.DB.Exec("UPDATE tasks SET deleted_at = NOW() WHERE id = $1", idStr)
	if err != nil {
		sendError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Проверка, была ли обновлена хоть одна строка
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		sendError(w, "Задача не найдена", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "deleted"}`))
}

// 🔹 PATCH /tasks/{id}/restore — Восстановление
func (h *TaskHandler) RestoreTask(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	_, err := h.DB.Exec("UPDATE tasks SET deleted_at = NULL WHERE id = $1", idStr)
	if err != nil {
		sendError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "restored"}`))
}

// 🔹 PATCH /tasks/{id}/archive — Архивация
func (h *TaskHandler) ArchiveTask(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	_, err := h.DB.Exec("UPDATE tasks SET archived_at = NOW() WHERE id = $1", idStr)
	if err != nil {
		sendError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "archived"}`))
}

// 🔹 PUT /tasks/{id} — Обновление
func (h *TaskHandler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	taskID, _ := strconv.Atoi(idStr)

	var t Task
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		sendError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	query := `
       UPDATE tasks 
       SET title=$1, 
          description=$2, 
          priority=$3, 
          column_id=$4::int, 
          position=$5,
          done_at = CASE 
             WHEN $4::int = $7::int THEN COALESCE(done_at, NOW()) 
             ELSE NULL 
          END,
          archived_at = CASE 
             WHEN $4::int = $7::int AND done_at < NOW() - INTERVAL '5 days' THEN NOW() 
             ELSE archived_at 
          END
       WHERE id=$6`

	_, err := h.DB.Exec(query,
		t.Title,       // $1
		t.Description, // $2
		t.Priority,    // $3
		t.ColumnID,    // $4
		t.Position,    // $5
		taskID,        // $6
		doneColumnID,  // $7
	)

	if err != nil {
		sendError(w, "SQL Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "updated"}`))
}
