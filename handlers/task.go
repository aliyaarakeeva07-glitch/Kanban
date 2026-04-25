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

// Укажи здесь ID колонки "Done" из своей базы данных
const doneColumnID = 3

// 🔹 GET /tasks?column_id=1 — Получить активные задачи
func (h *TaskHandler) GetTasks(w http.ResponseWriter, r *http.Request) {
	colIDStr := r.URL.Query().Get("column_id")
	colID, err := strconv.Atoi(colIDStr)
	if err != nil {
		http.Error(w, "Invalid column_id", 400)
		return
	}

	rows, err := h.DB.Query(`
		SELECT id, column_id, title, description, priority, position, done_at 
		FROM tasks 
		WHERE column_id=$1 AND deleted_at IS NULL AND archived_at IS NULL 
		ORDER BY position ASC`, colID)

	if err != nil {
		http.Error(w, err.Error(), 500)
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
		http.Error(w, "Invalid JSON", 400)
		return
	}

	var maxPos int
	h.DB.QueryRow("SELECT COALESCE(MAX(position), 0) FROM tasks WHERE column_id=$1", t.ColumnID).Scan(&maxPos)

	err := h.DB.QueryRow(`
		INSERT INTO tasks (column_id, title, description, priority, position)
		VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		t.ColumnID, t.Title, t.Description, t.Priority, maxPos+1).Scan(&t.ID)

	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(t)
}

// 🔹 DELETE /tasks/{id} — Мягкое удаление
func (h *TaskHandler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	_, err := h.DB.Exec("UPDATE tasks SET deleted_at = NOW() WHERE id = $1", idStr)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Write([]byte(`{"status": "deleted"}`))
}

// 🔹 PATCH /tasks/{id}/restore — Восстановление из корзины
func (h *TaskHandler) RestoreTask(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	_, err := h.DB.Exec("UPDATE tasks SET deleted_at = NULL WHERE id = $1", idStr)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Write([]byte(`{"status": "restored"}`))
}

// 🔹 PATCH /tasks/{id}/archive — Ручная архивация
func (h *TaskHandler) ArchiveTask(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	_, err := h.DB.Exec("UPDATE tasks SET archived_at = NOW() WHERE id = $1", idStr)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Write([]byte(`{"status": "archived"}`))
}

// 🔹 PUT /tasks/{id} — Обновление + Логика Done + Авто-архив
func (h *TaskHandler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	taskID, _ := strconv.Atoi(idStr)

	var t Task
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		http.Error(w, "Invalid JSON", 400)
		return
	}

	// Добавили ::int к $4 и $7, чтобы убрать ошибку со скриншота 57
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
		http.Error(w, "SQL Error: "+err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "updated"}`))
}

// SQL магия:
// 1. Если попали в Done (ID 3) — ставим done_at. Если ушли — зануляем.
// 2. Если уже в Done > 5 дней — ставим archived_at.
