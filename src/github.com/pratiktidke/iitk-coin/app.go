package main

import (
	"net/http"
	"github.com/pratiktidke/iitk-coin/authPackage"
	transac "github.com/pratiktidke/iitk-coin/transacPackage"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	http.HandleFunc("/signup", authPackage.SignUp)
	http.HandleFunc("/signin", authPackage.SignIn)
	http.HandleFunc("/secretpage", authPackage.Secretpage)
	http.HandleFunc("/awardCoins", transac.AwardCoins)
	http.HandleFunc("/transferCoins", transac.Transfer)
	http.HandleFunc("/checkBalance", transac.CheckBalance)
	http.ListenAndServe(":3000", nil)
}
