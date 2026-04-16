package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

func main() {
	// В dbname пишем Kanban-Board (как в pgAdmin)
	// В password пишем 1234 (или тот, что вводили при установке слоника)
	connStr := "host=localhost port=5432 user=postgres password=aliya020507 dbname=Kanban-Board sslmode=disable"

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Проверяем само соединение
	err = db.Ping()
	if err != nil {
		fmt.Println("Ошибка: база не пустила нас. Проверь пароль!")
		log.Fatal(err)
	}

	fmt.Println("Успех! База данных Kanban-Board подключена.")
}
