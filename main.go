/*
https://blog.rebrandly.com/the-history-of-url-shorteners/ 
https://www.educative.io/courses/grokking-the-system-design-interview/m2ygV4E81AR
https://blog.garstasio.com/you-dont-need-jquery/selectors/ 
https://www.restapitutorial.com/httpstatuscodes.html
https://www.digitalocean.com/community/tutorials/how-to-install-and-use-postgresql-on-ubuntu-18-04
https://www.calhoun.io/connecting-to-a-postgresql-database-with-gos-database-sql-package/
https://regexr.com/
https://www.alexedwards.net/blog/how-to-rate-limit-http-requests
*/

package main

import (
	"os"
	"fmt"
	"log"
	"regexp"
	"net/http"
	"database/sql"
	"html/template"
	"encoding/json"
	_ "github.com/lib/pq"
	"github.com/gorilla/mux"
)

var (
	Port string = os.Getenv("PORT")
	key = []byte(os.Getenv("COOKIE_KEY"))
	conn string = os.Getenv("DATABASE_URL")
	db *sql.DB = dbConn()
	urlValid *regexp.Regexp = regexp.MustCompile("^https:\\/\\/tinyurl.com\\/[A-Za-z0-9]{5}$")
)

func dbConn() *sql.DB {
	db, err := sql.Open("postgres", conn)
	if err != nil {
		panic(err)		
	}

	err = db.Ping()
	if err != nil {
		panic(err)
	}
	return db
}

func createTable(db *sql.DB) {
	query := `CREATE TABLE IF NOT EXISTS tinywhacks (
                   url varchar (255) NOT NULL,
                   status varchar (255) NOT NULL,
                   whack_date timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
                  );`
	_, err := db.Exec(query)
	if err != nil {
		panic(err)
	}
}

func main() {
	// setup
	defer db.Close()
	createTable(db)
	r:= mux.NewRouter()
	
	// static file serving
	cssHandler := http.FileServer(http.Dir("public/"))
	r.PathPrefix("/public/").Handler(http.StripPrefix("/public/", cssHandler))
	
	// landing page
	r.HandleFunc("/", landing).Methods("GET")
	
	// url processing
	r.HandleFunc("/register_url",registerUrl).Methods("POST")
	http.ListenAndServe(":"+Port, limit(r))
}

func landing(w http.ResponseWriter, r *http.Request) {
	tmp := template.Must(template.ParseFiles("public/index.html"))
	tmp.Execute(w,nil)
}

type Entry struct {
	Url    string `json:"url"`
	Status int    `json:"status"` 
}

func registerUrl(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		// get the data
		var entry Entry
		err := json.NewDecoder(r.Body).Decode(&entry)

		if err != nil {
			log.Printf("Unable to decode data in r.Body: %v", r.Body)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}


		// validate url
		isMatch := urlValid.FindStringSubmatch(entry.Url)
		if isMatch == nil {
			log.Printf("Unable validate url for data: %v", entry.Url)
			http.Error(w, "", http.StatusBadRequest)
			return
		}


		// validate status, everything from -1 to 599
		if (entry.Status > 599) || (entry.Status < -1) {
			log.Printf("Unable to validate status for data: %v", entry.Status)
			http.Error(w, "", http.StatusBadRequest)
			return
		}

		// register data
		query := fmt.Sprintf("INSERT into tinywhacks VALUES ($1, $2);")
		_, err = db.Exec(query, entry.Url, entry.Status)
		if err != nil {
			log.Printf("Unable to register validated data: %v", entry)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

