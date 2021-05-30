package main

import (
	"database/sql"
	"fmt"
	"strconv"

	_ "github.com/mattn/go-sqlite3"
)

type student struct {
	Name    string
	Roll_No int
}

func Insert_Row(Student *student, database *sql.DB) {
	statement, _ := database.Prepare("INSERT INTO user (name, roll_no) VALUES (?, ?)")
	statement.Exec(Student.Name, Student.Roll_No)
}
func main() {
	database, _ := sql.Open("sqlite3", "./DB.db")
	statement, _ := database.Prepare("CREATE TABLE IF NOT EXISTS user (id INTEGER PRIMARY KEY, name TEXT, roll_no INTEGER)")
	statement.Exec()

	Student := student{"drogon", 111}
	Insert_Row(&Student, database) // function to insert row in database

	// code to print table(user) in DB
	rows, _ := database.Query("SELECT id, name, roll_no FROM user")
	var id int
	var name string
	var roll_no int
	for rows.Next() {
		rows.Scan(&id, &name, &roll_no)
		fmt.Println(strconv.Itoa(id)+": "+name, roll_no)
	}
}
