package transac

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"task4/authPackage"

	"github.com/dgrijalva/jwt-go"
)

var channel chan int
var dbase *sql.DB
var err error

func init() {
	//for locking the concurrent executions while one transaction is on
	channel = make(chan int, 1)
	channel <- (1)

	dbase, err = sql.Open("sqlite3", "./database.db")
	if err != nil {
		fmt.Println(err)
		return
	}

	_, err = dbase.Exec("CREATE TABLE IF NOT EXISTS transacHistorys (id INTEGER PRIMARY KEY, sender TEXT, receiver TEXT, amount REAL, description TEXT, date_and_time TEXT)")
	if err != nil {
		fmt.Println(err)
		return
	}
	_, err = dbase.Exec("CREATE TABLE IF NOT EXISTS redeemTbl (id INTEGER PRIMARY KEY, item TEXT, roll_no TEXT, price REAL, status TEXT)")
	if err != nil {
		fmt.Println(err)
		return
	}
	_, err = dbase.Exec("CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY, roll_no TEXT, password TEXT, coins REAL, role TEXT, noOfEvents INTEGER)")
	if err != nil {
		fmt.Println(err)
		return
	}
}

// for getting RollNo of logged user
func FindUserFromTokenString(tknStr string) (string, error) {
	claim := &authPackage.CustomClaims{}
	tkn, _ := jwt.ParseWithClaims(tknStr, claim, func(t *jwt.Token) (interface{}, error) {
		return authPackage.JwtKey, nil
	})
	if tkn.Valid {
		return claim.Roll_no, nil
	}
	return "", errors.New("token not Valid")
}

//needed to decide how much tax to apply on transfers
func FindRole(user string) string {

	rows, _ := dbase.Query("SELECT role FROM users WHERE roll_no=?", user)
	var role string
	for rows.Next() {
		_ = rows.Scan(&role)
	}
	return role

}

// for checking eligibility to transfer coins
func FindEventsParticipated(user string) (int, error) {
	rows, err := dbase.Query("SELECT noOfEvents FROM users WHERE roll_no = ?", user)

	if err != nil {
		fmt.Println(err)
		return -1, err
	}

	var noOfEvents int
	for rows.Next() {
		_ = rows.Scan(&noOfEvents)
	}

	return noOfEvents, nil
}

//transaction endpoint handler functions

func AwardCoins(w http.ResponseWriter, r *http.Request) {

	c, err := r.Cookie("token")
	if err != nil {
		w.Write([]byte("Please Login First "))
		return
	}

	admin, err := FindUserFromTokenString(c.Value)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	A, err := strconv.Atoi(r.FormValue("amount"))
	if err != nil {
		fmt.Println(err)
		return
	}

	amount := float64(A)
	receiver := r.FormValue("awardTo")

	if !authPackage.UserExists(receiver) {
		w.Write([]byte("Receiver not registered"))
		return
	}

	roleAdmin, roleReceiver := FindRole(admin), FindRole(receiver)

	if roleAdmin == "CTM" && roleReceiver != "CTM" { // CTM here denotes Core Team Member

		<-channel

		tx, err := dbase.Begin() //transaction begins
		if err != nil {
			fmt.Println(err)
			channel <- (1)
			tx.Rollback()
			return
		}

		_, err = tx.Exec("UPDATE users SET coins = coins + ?, noOfEvents = noOfEvents + 1 WHERE roll_no = ?", amount, receiver)
		if err != nil {
			fmt.Println(err)
			channel <- (1)
			tx.Rollback()
			return
		}

		_, err = tx.Exec("INSERT INTO  transacHistorys (sender, receiver, amount, description, date_and_time) VALUES (?,?,?,?,datetime('now'))", admin, receiver, amount, "Awarded")
		if err != nil {
			fmt.Println(err)
			channel <- (1)
			tx.Rollback()
			return
		}

		tx.Commit() //transaction commited

		channel <- (1)

		return
	}

	w.Write([]byte("Either admin not CTM or receiver is CTM"))
}

func Transfer(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("token")
	if err != nil {
		w.Write([]byte("Please Login First "))
		return
	}

	sender, err := FindUserFromTokenString(c.Value)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	A, err := strconv.Atoi(r.FormValue("amount"))
	if err != nil {
		fmt.Println(err)
		return
	}

	amount := float64(A)
	receiver := r.FormValue("sendTo")

	if !authPackage.UserExists(receiver) {
		w.Write([]byte("Reciever not registered"))
		return
	}

	roleOfSender, roleOfReceiver := FindRole(sender), FindRole(receiver)

	if roleOfReceiver != "CTM" {

		<-channel

		tx, err := dbase.Begin() //transaction begins
		if err != nil {
			fmt.Println(err)
			channel <- (1)
			tx.Rollback()
			return
		}

		res, err := tx.Exec("UPDATE users SET coins = coins - ? WHERE roll_no = ? AND coins - ? >= 0", amount, sender, amount)
		if err != nil {
			fmt.Println(err)
			channel <- (1)
			tx.Rollback()
			return
		}

		affectedRows, err := res.RowsAffected()
		if err != nil {
			fmt.Println(err)
			channel <- (1)
			tx.Rollback()
			return
		}

		if affectedRows != 1 {
			w.Write([]byte("insufficient balance"))
			channel <- (1)
			tx.Rollback()
			return
		}

		if roleOfReceiver == roleOfSender {

			_, err = tx.Exec("UPDATE users SET coins = coins + ? WHERE roll_no = ?", 0.98*amount, receiver)
			if err != nil {
				fmt.Println(err)
				channel <- (1)
				tx.Rollback()
				return
			}

		} else {
			_, err = tx.Exec("UPDATE users SET coins = coins + ? WHERE roll_no = ?", 0.67*amount, receiver)
			if err != nil {
				fmt.Println(err)
				channel <- (1)
				tx.Rollback()
				return
			}

		}

		_, err = tx.Exec("INSERT INTO transacHistorys (sender, receiver, amount, description,date_and_time) VALUES(?,?,?,?,datetime('now'))", sender, receiver, amount, "Transfer")
		if err != nil {
			fmt.Println(err)
			channel <- (1)
			tx.Rollback()
			return
		}

		tx.Commit() //transaction commited

		channel <- (1)

	} else {

		w.Write([]byte("Transaction not permitted"))
		return

	}
}

func CheckBalance(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("token")
	if err != nil {
		w.Write([]byte("Please Login First "))
		return
	}
	if !authPackage.AuthenticateUser(c) {
		w.Write([]byte("User not logged in"))
		return
	}

	user, err := FindUserFromTokenString(c.Value)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	rows, err := dbase.Query("SELECT coins FROM users WHERE roll_no=?", user)
	if err != nil {
		fmt.Println(err)
		return
	}

	var coins float64
	for rows.Next() {
		_ = rows.Scan(&coins)
	}

	w.Write([]byte(strconv.FormatFloat(coins, 'f', -1, 64)))

}

func Redeem(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("token")
	if err != nil {
		w.Write([]byte("Please Login First "))
		return
	}

	user, err := FindUserFromTokenString(c.Value)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	A, _ := strconv.Atoi(r.FormValue("price"))
	if err != nil {
		fmt.Println(err)
		return
	}

	price := float64(A)
	itemName := r.FormValue("item-name")

	<-channel

	tx, err := dbase.Begin() //transaction begins

	if err != nil {
		fmt.Println(err)
		channel <- (1)
		return
	}

	_, err = tx.Exec("INSERT INTO redeemTbl (item, roll_no, price, status) VALUES(?,?,?,?)", itemName, user, price, "pending")
	if err != nil {
		fmt.Println(err)
		channel <- (1)
		tx.Rollback()
		return
	}

	tx.Commit() //transaction ends

	channel <- (1)
}

func UpdateRequestStatus(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.FormValue("request_id"))
	newStatus := r.FormValue("status")

	<-channel
	tx, err := dbase.Begin()
	if err != nil {
		fmt.Println(err)
		channel <- (1)
		tx.Rollback()
		return
	}

	rows, err := tx.Query("SELECT roll_no, price, status FROM redeemTbl WHERE id = ?", id)
	if err != nil {
		fmt.Println(err)
		channel <- (1)
		tx.Rollback()
		return
	}

	var user string
	var price float64
	var curStatus string
	for rows.Next() {
		err = rows.Scan(&user, &price, &curStatus)
		if err != nil {
			fmt.Println(err)
			channel <- (1)
			tx.Rollback()
			return
		}
	}

	if curStatus != "pending" {
		w.Write([]byte("status is not pending"))
		channel <- (1)
		tx.Rollback()
		return
	}

	row, err := tx.Query("SELECT coins FROM users WHERE roll_no = ?", user)
	if err != nil {
		fmt.Println(err)
		channel <- (1)
		tx.Rollback()
		return
	}

	var balance float64

	for row.Next() {
		row.Scan(&balance)
		if err != nil {
			fmt.Println(err)
			channel <- (1)
			tx.Rollback()
			return
		}
	}

	if balance >= price {

		if newStatus == "accepted" {
			_, err = tx.Exec("UPDATE users SET coins = coins - ? WHERE roll_no = ?", price, user)
			if err != nil {
				fmt.Println(err)
				channel <- (1)
				tx.Rollback()
				return
			}
		}
		_, err = tx.Exec("UPDATE redeemTbl SET status = ? WHERE id = ?", newStatus, id)
		if err != nil {
			fmt.Println(err)
			channel <- (1)
			tx.Rollback()
			return
		}

	} else {

		_, err = tx.Exec("UPDATE redeemTbl SET status = ? WHERE id = ?", "rejected", id)
		if err != nil {
			fmt.Println(err)
			channel <- (1)
			tx.Rollback()
			return
		}

	}
	tx.Commit() //transaction ends
	channel <- (1)
}
