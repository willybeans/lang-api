package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-rod/rod"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID        int    // `json:"id"` //was int
	Username  string // `json:"username"`
	Password  string // `json:"-"`
	Time      int
	Firstname string
	Lastname  string
	Email     string
	Age       int
}

var db *sql.DB

const (
	host     = "localhost"
	port     = 5432
	user     = "postgres"
	password = "password"
	dbname   = "lang_api"
)

func main() {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	// Connect to the PostgreSQL database
	var err error

	db, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close() // defer pushes function call onto list, which is called after the surrounding function is complete.
	// this is commonly used to simply functions that perform various cleanup tasks, ie, closing the db here

	err = db.Ping()
	if err != nil {
		panic(err)
	}
	// Initialize the Chi router
	router := chi.NewRouter()

	// Middleware
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	// API routes
	router.Get("/healthcheck", healthcheck)
	router.Get("/scrape", scrape)
	router.Post("/register", registerHandler)
	router.Post("/login", loginHandler)

	// Run the server
	err = http.ListenAndServe(":8080", router)
	if err != nil {
		log.Fatal(err)
	}
}

func healthcheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Healthcheck!"})
}

func scrape(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	// https://www.spiegel.de/international/world/escalating-violence-radical-settlers-on-the-west-bank-see-an-opportunity-a-9499f824-9b39-4739-b6db-36772bc2bb99
	// Decode the request body into the User struct
	// var url string
	url := r.URL.Query().Get("url")
	fmt.Println("url =>", url)
	// err := json.NewDecoder(r.Body)
	// if err != nil {
	// 	http.Error(w, "Invalid request", http.StatusBadRequest)
	// 	return
	// }

	// Launch a new browser with default options, and connect to it.

	browser := rod.New().MustConnect()

	// close it after main process ends.
	defer browser.MustClose()

	// Create a new page
	page := browser.MustPage(url).MustWaitStable()
	//DER SPIEGEL EXAMPLE
	//heading
	// fmt.Println(page.MustElement("#Inhalt > article > header > div > div").MustEval(`() => this.innerText`).String())
	heading := page.MustElement("#Inhalt > article > header > div > div").MustEval(`() => this.innerText`).String()
	// fmt.Printf(heading)

	// main := page.MustElement("main").MustEval(`() => {
	// 	return this.innerText
	// 	}`)
	// fmt.Printf("%+v\n", main)

	main := page.MustElements("header")
	fmt.Printf("%+v\n", main)

	// main := page.MustSearch("main").MustEval(`() => {
	// 	return this
	// 	}`)
	// fmt.Printf("____  TEST  ____ \n%+v\n %T", main, main)
	// 	var res
	// 	err := json.Unmarshal([]byte(str), &res)
	// 	fmt.Println(err)
	// 	fmt.Println(res)
	// 	for _, m := range main {

	//     // m is a map[string]interface.
	//     // loop over keys and values in the map.
	//     for k, v := range m {
	//         fmt.Println(k, "value is", v)
	//     }
	// }

	// header := page.MustElement("heading")
	// fmt.Printf("%+v\n", header)

	text := page.MustElements("p")
	fmt.Printf("%+v\n %T", text, text)
	//body
	// fmt.Println(page.MustElement("#Inhalt > article > div.relative > section.relative > div > div").MustEval(`() => this.innerText`).String())
	body := page.MustElement("#Inhalt > article > div.relative > section.relative > div > div").MustEval(`() => this.innerText`).String()
	// fmt.Printf(body)

	json.NewEncoder(w).Encode(map[string]string{"heading": heading, "body": body})
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	var user User

	// Decode the request body into the User struct
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Hash the user password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	// get ID and time
	var id int

	rows, err := db.Query("SELECT nextval('id_seq');")
	if err != nil {
		s := err.Error()
		fmt.Printf("Failed to retreive sequence ID for user id\n s: %v", s)
		http.Error(w, "Failed to register user", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	for rows.Next() {
		err := rows.Scan(&id)
		if err != nil {
			log.Fatal(err)
		}
		log.Println(id)
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	now := time.Now().Unix()

	// Save the user to the database
	_, err = db.Exec("INSERT INTO users (username, password, time_created, id, age, first_name, last_name, email) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)", user.Username, hashedPassword, now, id, user.Age, user.Firstname, user.Lastname, user.Email)
	if err != nil {
		s := err.Error()
		fmt.Printf("type: %T; value: %q\n", s, s)
		http.Error(w, "Failed to register user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	var user User

	// Decode the request body into the User struct
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Retrieve the user from the database
	row := db.QueryRow("SELECT * FROM users WHERE username = $1", user.Username)
	err = row.Scan(&user.ID, &user.Username, &user.Password)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Compare the provided password with the hashed password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(user.Password))
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Login successful"})
}
