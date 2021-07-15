package main

import (
	"net/http"
	"task4/authPackage"
	transac "task4/transacPackage"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	http.HandleFunc("/signup", authPackage.SignUp)
	http.HandleFunc("/signin", authPackage.SignIn)
	http.HandleFunc("/awardCoins", transac.AwardCoins)
	http.HandleFunc("/transferCoins", transac.Transfer)
	http.HandleFunc("/checkBalance", transac.CheckBalance)
	http.HandleFunc("/redeem", transac.Redeem)
	http.HandleFunc("/handleRequest", transac.UpdateRequestStatus)
	http.ListenAndServe(":3000", nil)
}
