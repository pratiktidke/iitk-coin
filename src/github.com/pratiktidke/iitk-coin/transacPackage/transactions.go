package transac

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"github.com/pratiktidke/iitk-coin/authPackage"

	"github.com/dgrijalva/jwt-go"
)

var channel chan int

func init() {
	channel = make(chan int, 1)
	channel <- (1)
}

var dbase *sql.DB

func init() {
	dbase, _ = sql.Open("sqlite3", "./database.db")
	_, err := dbase.Exec("CREATE TABLE IF NOT EXISTS transacHistorys (id INTEGER PRIMARY KEY, sender TEXT, receiver TEXT, amount REAL, description TEXT, date_and_time TEXT)")
	if err != nil {
		fmt.Println(err)
		return
	}
}
func FindUserFromToken(tknStr string) string {
	claim := &authPackage.CustomClaims{}
	tkn, _ := jwt.ParseWithClaims(tknStr, claim, func(t *jwt.Token) (interface{}, error) {
		return authPackage.JwtKey, nil
	})
	if tkn.Valid {
		return claim.Roll_no
	}
	return "N"
}

func FindRole(user string) string {

	rows, _ := dbase.Query("SELECT role FROM users WHERE roll_no=?", user)
	var role string
	for rows.Next() {
		_ = rows.Scan(&role)
	}
	return role

}

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

func AwardCoins(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("token")
	if err != nil {
		w.Write([]byte("Please Login First "))
		return
	}
	admin := FindUserFromToken(c.Value)
	A, _ := strconv.Atoi(r.FormValue("amount"))
	amount := float64(A)
	receiver := r.FormValue("awardTo")

	fmt.Println(receiver)
	if admin == "N" {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("User not logged in"))
		return
	}
	if !authPackage.UserExists(receiver) {
		w.Write([]byte("Receiver not registered"))
		return
	}

	roleAdmin, roleReceiver := FindRole(admin), FindRole(receiver)

	if roleAdmin == "CTM" && roleReceiver != "CTM" { // CTM here denotes Core Team Member
		<-channel
		tx, _ := dbase.Begin()

		res, err1 := tx.Exec("UPDATE users SET coins = coins + ?, noOfEvents = noOfEvents + 1 WHERE roll_no = ?", amount, receiver)

		affectedRows, err2 := res.RowsAffected()

		if affectedRows != 1 || err1 != nil || err2 != nil {
			tx.Rollback()
			channel <- (1)
			return
		}

		res, err1 = tx.Exec("INSERT INTO  transacHistorys (sender, receiver, amount, description, date_and_time) VALUES (?,?,?,?,datetime('now'))", admin, receiver, amount, "Awarded")

		affectedRows, err2 = res.RowsAffected()

		if affectedRows != 1 || err1 != nil || err2 != nil {
			tx.Rollback()
			channel <- (1)
			return
		}
		tx.Commit()
		channel <- (1)

		return
	}

	w.Write([]byte("Either admin not CTM or receiver is CTM"))
}

func Transfer(w http.ResponseWriter, r *http.Request) {
	c, _ := r.Cookie("token")
	sender := FindUserFromToken(c.Value)
	A, _ := strconv.Atoi(r.FormValue("amount"))
	amount := float64(A)
	receiver := r.FormValue("sendTo")

	if sender == "N" {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("User not logged in"))
	}
	if !authPackage.UserExists(receiver) {
		w.Write([]byte("Reciever not registered"))
		return
	}

	roleOfSender, roleOfReceiver := FindRole(sender), FindRole(receiver)

	if roleOfReceiver != "CTM" {
		<-channel
		tx, _ := dbase.Begin()
		res, err1 := tx.Exec("UPDATE users SET coins = coins - ? WHERE roll_no = ? AND coins - ? >= 0", amount, sender, amount)

		affectedRows, err2 := res.RowsAffected()

		if affectedRows != 1 || err1 != nil || err2 != nil {
			w.Write([]byte("insufficient balance or some other error"))
			tx.Rollback()
			channel <- (1)
			return
		}
		if roleOfReceiver == roleOfSender {
			res, err1 = tx.Exec("UPDATE users SET coins = coins + ? WHERE roll_no = ?", 0.98*amount, receiver)
		} else {
			res, err1 = tx.Exec("UPDATE users SET coins = coins + ? WHERE roll_no = ?", 0.67*amount, receiver)
		}
		affectedRows, err2 = res.RowsAffected()

		if affectedRows != 1 || err1 != nil || err2 != nil {
			w.Write([]byte("some other error"))
			tx.Rollback()
			channel <- (1)
			return
		}

		res, err1 = tx.Exec("INSERT INTO transacHistorys (sender, receiver, amount, description,date_and_time) VALUES(?,?,?,?,datetime('now'))", sender, receiver, amount, "Transfer")

		affectedRows, err2 = res.RowsAffected()

		if affectedRows != 1 || err1 != nil || err2 != nil {
			w.Write([]byte("some other error"))
			tx.Rollback()
			return
		}
		tx.Commit()
		channel <- (1)

	} else {
		w.Write([]byte("Transaction not permitted"))
		return
	}
}

func CheckBalance(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("token")

	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if !authPackage.AuthenticateUser(c) {
		w.Write([]byte("User not logged in"))
		return
	}

	user := FindUserFromToken(c.Value)

	rows, err := dbase.Query("SELECT coins FROM users WHERE roll_no = ?", user)

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
