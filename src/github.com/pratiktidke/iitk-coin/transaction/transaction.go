package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// var channel chan int

// func init() {
// 	channel = make(chan int, 1)
// 	channel <- (1)
// }

func accountExists(tx *sql.DB, roll_no string) (int, bool, error) {
	rows, err := tx.Query("SELECT coins FROM wallet WHERE rollno = ?", roll_no)
	if err != nil {
		return -1, false, err
	}

	cnt := 0
	var coins int
	for rows.Next() {
		cnt++
		rows.Scan(&coins)
	}
	if cnt == 1 {
		return coins, true, nil
	} else {
		return 0, false, nil
	}
}
func awardCoinsFunc(w http.ResponseWriter, r *http.Request) {
	//<-channel
	// validating request method
	if r.Method != "POST" {
		w.WriteHeader(http.StatusBadRequest)
		//channel <- (1)
		return
	}

	dbase, _ := sql.Open("sqlite3", "./userWallet.db")
	dbase.Exec("cache=shared;")
	dbase.Exec("PRAGMA read_uncommitted = true")
	// for closing database after all the code in AwardCoins func is executed

	_, accExists, err := accountExists(dbase, r.FormValue("roll-no"))
	if err != nil {
		fmt.Println(err)
		return
	}
	if !accExists {
		dbase.Exec("INSERT INTO wallet (rollno, coins) VALUES(?,?)", r.FormValue("roll-no"), 0)
	}

	tx, _ := dbase.Begin()
	Amount, _ := strconv.Atoi((r.FormValue("amount")))
	res, _ := tx.Exec("UPDATE wallet SET coins = coins + ? WHERE rollno = ?", Amount, r.FormValue("roll-no"))

	time.Sleep(10 * (time.Second))

	affectedRows, _ := res.RowsAffected()
	if affectedRows != 1 {
		tx.Rollback()
		return
	}
	// if accExists {
	// 	addedCoins, _ := strconv.Atoi(r.FormValue("coins"))
	// 	updatedCoins := coins + addedCoins
	// 	tx.Exec("UPDATE wallet SET coins=? WHERE rollno = ?", updatedCoins, r.FormValue("roll-no"))
	// } else {
	// 	coinsToAward, _ := strconv.Atoi(r.FormValue("coins"))
	// 	tx.Exec("INSERT INTO wallet (rollno, coins) VALUES(?,?)", r.FormValue("roll-no"), coinsToAward)
	// }
	tx.Commit()
	//channel <- (1)
}

func viewBalanceFunc(w http.ResponseWriter, r *http.Request) {
	//<-channel
	dbase, _ := sql.Open("sqlite3", "./userWallet.db")
	// dbase.Exec("cache=shared;")
	// dbase.Exec("PRAGMA read_uncommitted = true")

	defer dbase.Close()

	var user string

	for _, v := range r.URL.Query() {
		user = v[0]
	}

	coins, accExists, err := accountExists(dbase, user)

	if err != nil {
		fmt.Println(err)
		//channel <- (1)
		return
	}
	if accExists {
		w.Write([]byte(strconv.Itoa(coins)))
	} else {
		w.Write([]byte("User not registered"))
	}
	//channel <- (1)
}

func transactionFunc(w http.ResponseWriter, r *http.Request) {
	//<-channel
	dbase, err := sql.Open("sqlite3", "./userWallet.db")
	if err != nil {
		fmt.Println("This is the error : 1")
		fmt.Println(err)
		return
	}
	dbase.Exec("cache=private;")
	dbase.Exec("PRAGMA read_uncommitted = true")

	user1, user2 := r.FormValue("user1"), r.FormValue("user2")
	amount, err := strconv.Atoi(r.FormValue("amount"))
	if err != nil {
		fmt.Println("This is the error : 2")
		fmt.Println(err)
		return
	}
	_, user1accExists, err := accountExists(dbase, user1)
	if err != nil {
		fmt.Println("this one")
		fmt.Println(err)
		return
	}
	if !user1accExists {
		dbase.Exec("INSERT INTO wallet (rollno, coins) VALUES (?,?)", user1, 0)
	}

	_, user2accExists, err := accountExists(dbase, user2)
	if err != nil {
		fmt.Println("this one")
		fmt.Println(err)
		return
	}
	if !user2accExists {
		dbase.Exec("INSERT INTO wallet (rollno, coins) VALUES (?,?)", user2, 0)
	}

	tx, err := dbase.Begin()
	if err != nil {
		fmt.Println("This is the error : 3")
		fmt.Println(err)
		tx.Rollback()
		return
	}
	time.Sleep(1 * time.Second)
	res, err := tx.Exec("UPDATE wallet SET coins = coins - ? WHERE rollno=? AND coins - ? >= 0 ", amount, user1, amount)
	if err != nil {
		fmt.Println("This is the error : 4")

		fmt.Println(err)
		tx.Rollback()
		return
	}
	affectedRows, err := res.RowsAffected()
	if err != nil {
		fmt.Println("This is the error : 5")
		fmt.Println(err)
		tx.Rollback()
		return
	}
	if affectedRows != 1 {
		w.Write([]byte("error occured"))
		tx.Rollback()

		return
	}
	res, err = tx.Exec("UPDATE wallet SET coins = coins + ? WHERE rollno = ?", amount, user2)
	if err != nil {
		fmt.Println("This is the error : 6")

		fmt.Println(err)
		tx.Rollback()
		return
	}
	affectedRows, err = res.RowsAffected()
	if err != nil {
		fmt.Println("This is the error : 7")
		fmt.Println(err)
		tx.Rollback()
		return
	}
	if affectedRows != 1 {
		w.Write([]byte("error occured"))
		tx.Rollback()

		return
	}
	tx.Commit()
	//channel <- (1)
}
func main() {
	//creating a database if not exist
	database, err := sql.Open("sqlite3", "./userWallet.db")
	if err != nil {
		fmt.Println(err)
		return
	}

	//creating a table if not exists
	statement, _ := database.Prepare("CREATE TABLE IF NOT EXISTS wallet (id INTEGER PRIMARY KEY, rollno TEXT, coins INTEGER)")
	statement.Exec()
	// stmt, err := database.Prepare("PRAGMA journal_mode=WAL;")
	// if err != nil {
	// 	fmt.Println(err)
	// }
	// stmt.Exec()
	database.Exec("PRAGMA journal_mode=DELETE;")
	//database.Exec("txlock=immediate;")
	fmt.Println("listening to the port 3000...")

	http.HandleFunc("/home", awardCoinsFunc)
	http.HandleFunc("/balance", viewBalanceFunc)
	http.HandleFunc("/transaction", transactionFunc)
	http.ListenAndServe(":3000", nil)
}
