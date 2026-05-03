package main

import (
	"database/sql"
	"log"
	"net/http"

	"kanban/handlers"
	"kanban/middleware"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/rs/cors" // Добавили библиотеку CORS
)

func main() {
	connStr := "host=localhost port=5432 user=postgres password=aliya020507 dbname=Kanban-Board sslmode=disable"

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal("DB error:", err)
	}
	defer db.Close()

	auth := &handlers.AuthHandler{DB: db}
	board := &handlers.BoardHandler{DB: db}
	column := &handlers.ColumnHandler{DB: db}
	task := &handlers.TaskHandler{DB: db}

	r := mux.NewRouter()

	// --- AUTH ---
	r.HandleFunc("/register", auth.Register).Methods("POST")
	r.HandleFunc("/login", auth.Login).Methods("POST")

	// --- BOARDS ---
	r.Handle("/boards", middleware.JWT(http.HandlerFunc(board.GetBoards))).Methods("GET")
	r.Handle("/boards", middleware.JWT(http.HandlerFunc(board.CreateBoard))).Methods("POST")
	r.Handle("/boards/{id}", middleware.JWT(http.HandlerFunc(board.DeleteBoard))).Methods("DELETE")
	r.Handle("/boards/{id}/members", middleware.JWT(http.HandlerFunc(board.GetMembers))).Methods("GET")

	// --- COLUMNS ---
	r.Handle("/columns", middleware.JWT(http.HandlerFunc(column.GetColumns))).Methods("GET")
	r.Handle("/columns", middleware.JWT(http.HandlerFunc(column.CreateColumn))).Methods("POST")
	r.Handle("/columns/{id}", middleware.JWT(http.HandlerFunc(column.DeleteColumn))).Methods("DELETE")
	r.Handle("/columns/{id}/restore", middleware.JWT(http.HandlerFunc(column.RestoreColumn))).Methods("PATCH")

	// --- TASKS (ЗАДАЧИ) ---
	r.Handle("/tasks", middleware.JWT(http.HandlerFunc(task.GetTasks))).Methods("GET")
	r.Handle("/tasks", middleware.JWT(http.HandlerFunc(task.CreateTask))).Methods("POST")
	r.Handle("/tasks/{id}", middleware.JWT(http.HandlerFunc(task.DeleteTask))).Methods("DELETE")
	r.Handle("/tasks/{id}", middleware.JWT(http.HandlerFunc(task.UpdateTask))).Methods("PUT")
	r.Handle("/tasks/{id}/archive", middleware.JWT(http.HandlerFunc(task.ArchiveTask))).Methods("PATCH")
	r.Handle("/tasks/{id}/restore", middleware.JWT(http.HandlerFunc(task.RestoreTask))).Methods("PATCH")

	// --- COMMENTS (КОММЕНТАРИИ) ---
	r.Handle("/comments", middleware.JWT(http.HandlerFunc(task.CreateComment))).Methods("POST")
	r.Handle("/comments", middleware.JWT(http.HandlerFunc(task.GetComments))).Methods("GET")
	r.Handle("/comments/{id}", middleware.JWT(http.HandlerFunc(task.DeleteComment))).Methods("DELETE")

	// --- НАСТРОЙКА CORS (Чтобы фронтенд мог подключиться) ---
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"}, // Разрешаем доступ с любого адреса
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	})

	// Оборачиваем наш роутер 'r' в cors-обработчик
	handler := c.Handler(r)

	log.Println("Server running on :8080")

	// Запускаем сервер через 'handler' вместо 'r'
	log.Fatal(http.ListenAndServe(":8080", handler))
}
