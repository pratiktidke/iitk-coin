package authPackage

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/mattn/go-sqlite3"
)

var JwtKey []byte
var dbase *sql.DB

type UserInfo struct {
	roll_no  string
	password string
	role     string
}
type CustomClaims struct {
	Roll_no string `json:"roll_no"`
	jwt.StandardClaims
}

func init() {
	JwtKey = []byte("This_is_the_key")
	var err error
	dbase, err = sql.Open("sqlite3", "./database.db")
	if err != nil {
		fmt.Println(err)
	}

	dbase.Exec("CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY, roll_no TEXT, password TEXT, coins REAL, role TEXT, noOfEvents INTEGER)")
}

func UserExists(roll_no string) bool {
	rows, _ := dbase.Query("SELECT password FROM users WHERE roll_no = ?", roll_no)

	affecRows := 0

	for rows.Next() {
		affecRows++
	}

	return affecRows != 0
}
func insertUserInDB(newUser *UserInfo) int {
	// res, _ := dbase.Exec("SELECT password FROM users WHERE roll_no = ?", newUser.roll_no)

	// affecRows, _ := res.RowsAffected()
	// fmt.Println(affecRows)

	userExists := UserExists(newUser.roll_no)

	if userExists {
		return 0
	} else {
		dbase.Exec("INSERT INTO users (roll_no, password, coins, role, noOfEvents) VALUES (?,?,?,?,?)", newUser.roll_no, newUser.password, 0, newUser.role, 0)
		return 1
	}

}

func validateUser(user *UserInfo) bool {

	rows, _ := dbase.Query("SELECT password FROM users WHERE roll_no = ?", user.roll_no)

	cnt := 0
	var expectedPassword string

	for rows.Next() {
		_ = rows.Scan(&expectedPassword)
		cnt++
	}
	fmt.Println(cnt)
	if cnt != 1 || (expectedPassword != user.password) {
		return false
	}
	return true
}

func AuthenticateUser(token *http.Cookie) bool {
	tknStr := token.Value
	claim := &CustomClaims{}
	tkn, _ := jwt.ParseWithClaims(tknStr, claim, func(t *jwt.Token) (interface{}, error) {
		return JwtKey, nil
	})
	return tkn.Valid
}
func makeJwtToken(user UserInfo) string {

	expirationTime := time.Now().Add(60 * time.Minute)
	claim := &CustomClaims{
		Roll_no: user.roll_no,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
		},
	}

	jwtTkn := jwt.NewWithClaims(jwt.SigningMethodHS256, claim)

	tknStr, _ := jwtTkn.SignedString(JwtKey)

	return tknStr
}

func SignUp(w http.ResponseWriter, r *http.Request) {

	newUser := UserInfo{r.FormValue("roll_no"), r.FormValue("password"), r.FormValue("role")}

	inserted := insertUserInDB(&newUser)

	if inserted == 1 {
		w.Write([]byte("You are signed up :)"))
	} else {
		w.Write([]byte("This Roll-no already exists:("))
	}
}

func SignIn(w http.ResponseWriter, r *http.Request) {

	user := UserInfo{r.FormValue("roll_no"), r.FormValue("password"), r.FormValue("role")}

	validated := validateUser(&user)

	if !validated {
		w.Write([]byte("invalid roll-no or password"))
		return
	}
	tknStr := makeJwtToken(user)
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    tknStr,
		Expires:  time.Now().Add(60 * time.Minute),
		HttpOnly: true,
	})
}

func Secretpage(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("token")

	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	authenticated := AuthenticateUser(c)

	if !authenticated {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("you need to login"))
		return
	}
	w.Write([]byte("welcome to the secretpage"))
}
