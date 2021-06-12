package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/mattn/go-sqlite3"
)

var jwtKey []byte
var tlp *template.Template

func init() {

	// initializing tlp to create a path for the .gohtml files in local directory
	tlp = template.Must(template.ParseGlob("C:\\Users\\admin\\Desktop\\Gofilestemp\\src\\trainingnow\\template/*.gohtml"))
	jwtKey = []byte("This_is_my_key")
}

// struct for student
type student struct {
	Roll_No  string `json:"roll-no"`
	Password string `json:"password"`
}

type claim struct {
	Roll_no string `json:"roll-no"`
	jwt.StandardClaims
}

func Insert_Row(Student *student, database *sql.DB) {
	statement, _ := database.Prepare("INSERT INTO user (roll_no, password) VALUES (?, ?)")
	statement.Exec(Student.Roll_No, Student.Password)
}

func loginformfunc(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		tlp.ExecuteTemplate(w, "loginform.gohtml", nil)
		return
	}
	database, _ := sql.Open("sqlite3", "./DB.db")
	P := r.FormValue("roll-no")
	fmt.Println(P)
	rows, err := database.Query("SELECT password FROM user WHERE roll_no=?", r.FormValue("roll-no"))

	if err != nil {
		fmt.Println("first one")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	var expected_password string
	cnt := 0
	for rows.Next() {
		cnt++
		err = rows.Scan(&expected_password)
		fmt.Println(expected_password)
		if err != nil || (expected_password != r.FormValue("password")) {
			fmt.Println(err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}
	if cnt > 0 {
		expirationTime := time.Now().Add(60 * time.Minute)
		Claim := &claim{
			Roll_no: r.FormValue("roll-no"),
			StandardClaims: jwt.StandardClaims{
				ExpiresAt: expirationTime.Unix(),
			},
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claim)
		tknstring, err := token.SignedString(jwtKey)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:    "token",
			Value:   tknstring,
			Expires: expirationTime,
			HttpOnly: true,
		})
		http.Redirect(w, r, "/home", http.StatusSeeOther)
	} else {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

}

func deregister(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("token")
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	tokenString := c.Value
	Claims := claim{}
	tkn, err := jwt.ParseWithClaims(tokenString, &Claims, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})

	if err != nil {
		fmt.Println(err)
		if err == jwt.ErrSignatureInvalid {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if !tkn.Valid {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	database, _ := sql.Open("sqlite3", "./DB.db")

	_, _ = database.Exec("DELETE FROM user WHERE roll_no=?", Claims.Roll_no)

}
func signupform(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		tlp.ExecuteTemplate(w, "signupform.gohtml", nil)
		return
	}

	database, _ := sql.Open("sqlite3", "./DB.db")
	statement, _ := database.Prepare("CREATE TABLE IF NOT EXISTS user (id INTEGER PRIMARY KEY, roll_no TEXT, password TEXT)")
	statement.Exec()

	Student := student{r.FormValue("roll-no"), r.FormValue("password")}

	// to find if an element with same roll_no already exists of not
	rows, _ := database.Query("SELECT id FROM user WHERE roll_no=?", Student.Roll_No)
	cnt := 0
	for rows.Next() {
		cnt++
	}
	if cnt > 0 {
		w.Write([]byte("Roll-No already exists"))
		return
	}
	// inserting the element in database
	Insert_Row(&Student, database)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
	// fmt.Println()
	// err := json.NewDecoder(r.Body).Decode(&Student)
	// if err != nil {
	// 	//fmt.Println(r.Body)
	// 	// if the structure of the body is wrong then return a error header
	// 	w.WriteHeader(http.StatusBadRequest)
	// 	return
	// }

}

func aftersub(w http.ResponseWriter, r *http.Request) {
	fmt.Println()
	tlp.ExecuteTemplate(w, "afterform.gohtml", nil)
}

func WelcomePage(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("token")
	if err != nil {
		fmt.Println(err)
		if err == http.ErrNoCookie {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	tknstring := c.Value
	Claim := &claim{}
	tkn, err := jwt.ParseWithClaims(tknstring, Claim, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil {
		fmt.Println(err)
		if err == jwt.ErrSignatureInvalid {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if !tkn.Valid {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	tlp.ExecuteTemplate(w, "welcome.gohtml", nil)
}

func signoutfunc(w http.ResponseWriter, r *http.Request) {
	c := http.Cookie{
		Name:   "token",
		MaxAge: -1}
	http.SetCookie(w, &c)
}
func main() {
	http.HandleFunc("/secretpage", WelcomePage)
	http.HandleFunc("/login", loginformfunc)
	http.HandleFunc("/home", aftersub)
	http.HandleFunc("/signup", signupform)
	http.HandleFunc("/signout", signoutfunc)
	http.HandleFunc("/deregister", deregister)
	http.ListenAndServe(":3000", nil)
}
